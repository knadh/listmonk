package models

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/preslavrachev/gomjml/mjml"
)

// compileMJMLCampaign compiles a campaign that uses an MJML wrapper template
// into a ready-to-execute Go HTML template.
//
// The campaign body is resolved to an inlineable fragment based on its content type:
//   - MJML: inlined directly as MJML components (e.g. <mj-text>, <mj-image>)
//   - Markdown: converted to HTML, then wrapped in <mj-raw>
//   - HTML / Richtext / Plain: wrapped in <mj-raw> as-is
//
// The fragment is substituted at the {{ template "content" . }} placeholder in
// the wrapper, producing a complete MJML document. gomjml renders that to HTML
// once (not per-subscriber). Go template syntax inside MJML text nodes is
// preserved verbatim by the renderer and resolved at send time per subscriber.
func compileMJMLCampaign(c *Campaign, f template.FuncMap) (*template.Template, error) {
	bodyFragment, err := mjmlBodyFragment(c)
	if err != nil {
		return nil, err
	}

	wrapper := c.TemplateBody
	if wrapper == "" {
		wrapper = bodyFragment
	} else {
		wrapper = strings.ReplaceAll(wrapper, `{{ template "content" . }}`, bodyFragment)
	}

	for _, r := range regTplFuncs {
		wrapper = r.regExp.ReplaceAllString(wrapper, r.replace)
	}

	htmlBody, err := mjml.Render(wrapper)
	if err != nil {
		return nil, fmt.Errorf("error compiling MJML: %v", err)
	}

	tpl, err := template.New(BaseTpl).Funcs(f).Parse(htmlBody)
	if err != nil {
		return nil, fmt.Errorf("error compiling MJML template: %v", err)
	}
	return tpl, nil
}

// renderMJML renders a complete MJML document string to HTML.
func renderMJML(body string) (string, error) {
	out, err := mjml.Render(body)
	if err != nil {
		return "", fmt.Errorf("error rendering MJML: %v", err)
	}
	return out, nil
}

// mjmlBodyFragment converts the campaign body to a fragment that can be
// inlined into an MJML wrapper at the {{ template "content" . }} placeholder.
func mjmlBodyFragment(c *Campaign) (string, error) {
	switch c.ContentType {
	case CampaignContentTypeMJML:
		// Already MJML components -- inline directly.
		return c.Body, nil

	case CampaignContentTypeMarkdown:
		// Convert Markdown to HTML, then wrap as raw HTML inside MJML.
		var buf bytes.Buffer
		if err := markdown.Convert([]byte(c.Body), &buf); err != nil {
			return "", fmt.Errorf("error converting Markdown for MJML: %v", err)
		}
		return "<mj-raw>" + buf.String() + "</mj-raw>", nil

	default:
		// HTML, Richtext, Plain -- plug in as raw HTML.
		return "<mj-raw>" + c.Body + "</mj-raw>", nil
	}
}
