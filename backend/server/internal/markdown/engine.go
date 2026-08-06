package markdown

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// OnboardingTemplateData defines the variables accessible inside the Markdown templates.
type OnboardingTemplateData struct {
	Username  string
	Email     string
	GiteaURL  string
	SystemURL string
}

// RenderGFM takes a raw Markdown template containing Go variables (e.g., {{.Username}}),
// injects the data, and returns fully parsed GitHub-Flavored HTML.
func RenderGFM(rawMarkdown string, data interface{}) (string, error) {
	// Step 1: Inject variables using Go's text/template
	t, err := template.New("markdown").Parse(rawMarkdown)
	if err != nil {
		return "", fmt.Errorf("failed to parse template variables: %w", err)
	}

	var mdBuffer bytes.Buffer
	if err := t.Execute(&mdBuffer, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Step 2: Configure Goldmark for GitHub Flavored Markdown (GFM)
	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM), // Enables tables, task lists, strikethrough
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Adds IDs to headers for anchor links
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithUnsafe(), // Allows embedded raw HTML inside the markdown if needed
		),
	)

	// Step 3: Parse the injected Markdown into HTML
	var htmlBuffer bytes.Buffer
	if err := mdParser.Convert(mdBuffer.Bytes(), &htmlBuffer); err != nil {
		return "", fmt.Errorf("failed to convert markdown to html: %w", err)
	}

	return htmlBuffer.String(), nil
}