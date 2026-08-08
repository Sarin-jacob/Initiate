package workers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/Sarin-jacob/Initiate/internal/agenthub"
	"github.com/Sarin-jacob/Initiate/internal/db"
)

// StartExpirationCron boots a background ticker that sweeps the database
func StartExpirationCron(database *gorm.DB, hub *agenthub.Hub) {
	ticker := time.NewTicker(1 * time.Hour)
	log.Println("Automated Expiration Cron initialized (runs every hour)")

	go func() {
		for {
			<-ticker.C
			processExpirations(database, hub)
		}
	}()
}

func processExpirations(database *gorm.DB, hub *agenthub.Hub) {
	var expiredUsers []db.User
	
	// Find users whose expiration date is in the past, and who are still ACTIVE
	err := database.Where("expires_at IS NOT NULL AND expires_at < ? AND status = ?", time.Now(), "ACTIVE").Find(&expiredUsers).Error
	if err != nil {
		log.Printf("[CRON] Failed to query expired users: %v", err)
		return
	}

	if len(expiredUsers) > 0 {
		log.Printf("[CRON] Sweeping %d expired accounts...", len(expiredUsers))
	}

	for _, user := range expiredUsers {
		log.Printf("[CRON] Account expired: Initiating automated deprovision for %s", user.Username)

		var accesses []db.UserAccess
		database.Where("user_id = ?", user.ID).Find(&accesses)

		hasFailures := false
		
		// Automated purges default to safe archiving (no hard purges)
		payload := map[string]interface{}{
			"username":    user.Username,
			"purge_home":  false,
			"purge_repos": false,
		}

		for _, access := range accesses {
			var target db.TargetServer
			if database.First(&target, "id = ?", access.TargetID).Error != nil {
				continue
			}

			// We specifically invoke the Agent's Soft Deprovision Macro
			if target.SoftDeprovisionMacroID != "" {
				var macro db.Macro
				if database.First(&macro, "id = ?", target.SoftDeprovisionMacroID).Error == nil {
					
					// Parse the JSON blocks
					var steps []map[string]string
					json.Unmarshal([]byte(macro.Steps), &steps)

					targetSuccess := true
					for _, step := range steps {
						event := fmt.Sprintf("%s:%s", step["module"], step["action"])
						log.Printf("[CRON] Executing %s on %s...", event, target.ID)
						
						res, err := hub.DispatchTaskSync(target.ID, event, payload, 30*time.Second)
						if err != nil || res.Status == "FAILED" {
							log.Printf("[CRON] Teardown failed on %s (Step: %s). Output: %v", target.ID, event, res.Output)
							targetSuccess = false
							break
						}
					}
					
					if !targetSuccess {
						hasFailures = true
						continue // Do not delete the access record if the agent failed
					}
				}
			}
			// Clean up access if successful or if no macro was required
			database.Delete(&access)
		}

		if hasFailures {
			database.Model(&user).Update("status", "DEPROVISION_FAILED")
			log.Printf("[CRON] Marked %s as DEPROVISION_FAILED (Requires manual intervention)", user.Username)
		} else {
			database.Model(&user).Update("status", "ARCHIVED")
			log.Printf("[CRON] Successfully archived %s", user.Username)
		}
	}
}