package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/config"
	"github.com/Sarin-jacob/Initiate/internal/db"
	"github.com/Sarin-jacob/Initiate/internal/gitea"
	"github.com/Sarin-jacob/Initiate/internal/mailer"
	"github.com/Sarin-jacob/Initiate/internal/markdown"
)

type CompleteOnboardingRequest struct {
	UserInputs map[string]string `json:"user_inputs"` // Captures the dynamic form!
}

func HandleCompleteOnboarding(database *gorm.DB, hub *agenthub.Hub, giteaClient *gitea.Client, emailer *mailer.Mailer, baseSystemURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CompleteOnboardingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		invite := r.Context().Value("invite").(db.Invitation)

		var user db.User
		if err := database.First(&user, "id = ?", invite.UserID).Error; err != nil {
			http.Error(w, "Associated user not found", http.StatusInternalServerError)
			return
		}

		// 1. Gracefully extract core fields if they exist in the dynamic inputs
		password, hasPassword := req.UserInputs["password"]
		sshKey, hasSSH := req.UserInputs["ssh_public_key"]

		var hashString string
		if hasPassword && password != "" {
			hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err == nil {
				hashString = string(hashBytes)
			}
		}

		// 2. Build the Unified Payload Dictionary (Bridge to Phase 4)
		// This map will look like: {"sys.username": "sarin", "user.password": "secret123"}
		payload := map[string]interface{}{
			"sys.username": user.Username,
			"sys.email":    user.Email,
			"sys.id":       user.ID,
		}
		// Inject all the user's dynamic inputs into the payload mapping
		for k, v := range req.UserInputs {
			payload[fmt.Sprintf("user.%s", k)] = v
		}

		var adminContext map[string]string
		if user.AdminContext != "" {
			json.Unmarshal([]byte(user.AdminContext), &adminContext)
			for k, v := range adminContext {
				payload[fmt.Sprintf("admin.%s", k)] = v
			}
		}

		// 3. UNIFIED PIPELINE EXECUTION
		var accesses []db.UserAccess
		database.Where("user_id = ?", user.ID).Find(&accesses)

		for _, access := range accesses {
			var target db.TargetServer
			if err := database.First(&target, "id = ?", access.TargetID).Error; err != nil {
				continue
			}

			if target.ProvisionMacroID == "" {
				database.Model(&access).Update("status", "ACTIVE")
				continue 
			}

			var macro db.Macro
			if err := database.First(&macro, "id = ?", target.ProvisionMacroID).Error; err != nil {
				database.Model(&access).Update("status", "FAILED")
				continue
			}

			var steps []MacroStep
			json.Unmarshal([]byte(macro.Steps), &steps)

			var executionTrace string
			pipelineSuccess := true
			for _, step := range steps {
				event := fmt.Sprintf("%s:%s", step.Module, step.Action)
				executionTrace += fmt.Sprintf("[PENDING] %s...\n", event)

				resolvedPayload := resolveStepParams(step.Params, payload)
				res, err := hub.DispatchTaskSync(target.ID, event, resolvedPayload, 30*time.Second)
				
				if err != nil {
					pipelineSuccess = false
					executionTrace += fmt.Sprintf("[ERROR] Network/Timeout: %v\n", err)
					break
				}
				
				if res.Status == "FAILED" {
					pipelineSuccess = false
					executionTrace += fmt.Sprintf("[FAILED] Edge output:\n%s\n", res.Output)
					break
				}
				
				executionTrace += fmt.Sprintf("[SUCCESS] Edge output:\n%s\n", res.Output)
			}

			// Save the trace and status
			updates := map[string]interface{}{
				"status": "ACTIVE",
				"execution_log": executionTrace,
			}
			if !pipelineSuccess {
				updates["status"] = "FAILED"
			}
			database.Model(&access).Updates(updates)
		}

		// 4. Finalize Onboarding
		updates := map[string]interface{}{
			"status": "ACTIVE",
		}
		if hashString != "" {
			updates["password_hash"] = hashString
		}
		if hasSSH {
			updates["ssh_public_key"] = sshKey
		}

		tx := database.Begin()
		tx.Model(&invite).Update("used_at", time.Now())
		tx.Model(&user).Updates(updates)
		tx.Commit()

		go func() {
			loginURL := config.App.GiteaExternalURL
			
			// 1. Fetch configured Welcome Email CMS page
			var welcomeSlug db.SystemSetting
			database.Where("key = ?", "welcome_email_slug").First(&welcomeSlug)

			subject := "Your Access is Provisioned!"
			bodyMD := "## Welcome aboard, {{.Username}}!\n\nYour systems are fully provisioned.\n\n[Go to Dashboard]({{.GiteaURL}})"

			var welcomePage db.Page
			if welcomeSlug.Value != "" && database.Where("slug = ?", welcomeSlug.Value).First(&welcomePage).Error == nil {
				subject = welcomePage.Title // Use CMS Title as Subject
				bodyMD = welcomePage.Content // Use CMS Body
			}

			// 2. Render Markdown to HTML
			emailData := markdown.OnboardingTemplateData{
				Username:  user.Username,
				Email:     user.Email,
				GiteaURL: loginURL, // Mapped so {{.GiteaURL}} works in the markdown
				SystemURL: config.App.BaseURL,
			}
			
			renderedHTML, err := markdown.RenderGFM(bodyMD, emailData)
			if err != nil || renderedHTML == "" {
				renderedHTML = fmt.Sprintf("<h2>Welcome %s!</h2><p>Your setup is complete.</p>", user.Username)
			}

			// 3. Buttonize the main link
			styledHref := fmt.Sprintf(`style="background-color: #18181b; color: #ffffff; padding: 10px 20px; text-decoration: none; border-radius: 6px; display: inline-block; font-weight: bold; margin: 10px 0 20px 0;" href="%s"`, loginURL)
			renderedHTML = strings.ReplaceAll(renderedHTML, fmt.Sprintf(`href="%s"`, loginURL), styledHref)

			// 4. Append injected Docs (Keep your existing doc recovery loop here!)
			if user.AdminContext != "" {
				var adminCtx map[string]string
				json.Unmarshal([]byte(user.AdminContext), &adminCtx)
				if docsJSON, ok := adminCtx["_injected_docs"]; ok && docsJSON != "" {
					var slugs []string
					json.Unmarshal([]byte(docsJSON), &slugs)
					if len(slugs) > 0 {
						renderedHTML += `<hr style="border: none; border-top: 1px solid #e4e4e7; margin: 30px 0;">`
						renderedHTML += `<h3>Your Assigned Documentation</h3>`
						for _, slug := range slugs {
							var doc db.Page
							if err := database.First(&doc, "slug = ?", slug).Error; err == nil {
								docURL := fmt.Sprintf("%s/?docs=%s", config.App.BaseURL, doc.Slug)
								renderedHTML += fmt.Sprintf(`<a href="%s" style="border: 2px solid #e4e4e7; color: #3f3f46; padding: 8px 16px; text-decoration: none; border-radius: 6px; display: inline-block; font-weight: bold; margin: 0 10px 10px 0;">%s</a>`, docURL, doc.Title)
							}
						}
					}
				}
			}

			if emailer != nil {
				emailer.SendHTML(user.Email, subject, renderedHTML)
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Provisioning pipelines completed."})
	}
}

var paramRegex = regexp.MustCompile(`\{\{(.*?)\}\}`)

// resolveStepParams takes the macro's bound parameters and fills them using the master context dictionary
func resolveStepParams(stepParams map[string]string, contextData map[string]interface{}) map[string]interface{} {
	resolved := make(map[string]interface{})

	for key, val := range stepParams {
		// Check if the value is a variable binding like {{sys.username}}
		matches := paramRegex.FindStringSubmatch(val)
		
		if len(matches) == 2 {
			varName := matches[1] // Extracts "sys.username"
			
			// Look up the extracted variable name in our master payload dictionary
			if actualValue, exists := contextData[varName]; exists {
				resolved[key] = actualValue
			} else {
				// Fallback if missing (prevents edge agent from receiving null/panicking)
				log.Printf("Warning: Missing context variable for %s", varName)
				resolved[key] = ""
			}
		} else {
			// It's a static value (e.g., the string "docker" or "/bin/bash")
			resolved[key] = val
		}
	}
	return resolved
}