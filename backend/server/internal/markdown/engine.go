package markdown

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// ServerInfo makes target data accessible to templates via {{range .Servers}}
type ServerInfo struct {
	Name    string
	Address string 
}

// OnboardingTemplateData defines variables for Markdown injection
type OnboardingTemplateData struct {
	Username  string
	Email     string
	GiteaURL  string
	SystemURL string
	Token     string // NEW: Pass token so admins can hyperlink between pages: /invite/{{.Token}}/page/ssh
	InviteURL string
	Servers   []ServerInfo
}

func RenderGFM(rawMarkdown string, data interface{}) (string, error) {
	t, err := template.New("markdown").Parse(rawMarkdown)
	if err != nil {
		return "", fmt.Errorf("failed to parse template variables: %w", err)
	}

	var mdBuffer bytes.Buffer
	if err := t.Execute(&mdBuffer, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithUnsafe(), // We allow Unsafe here to parse it...
		),
	)

	var htmlBuffer bytes.Buffer
	if err := mdParser.Convert(mdBuffer.Bytes(), &htmlBuffer); err != nil {
		return "", fmt.Errorf("failed to convert markdown to html: %w", err)
	}

	// ... But we strictly sanitize it here. 
	// UGCPolicy allows standard styling and links, but strips <script>, <style>, and on-click handlers.
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Globally()
	policy.AllowAttrs("style").Globally()
	policy.AllowAttrs("target").OnElements("a")
	safeHTML := policy.SanitizeBytes(htmlBuffer.Bytes())

	return string(safeHTML), nil
}