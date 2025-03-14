package web

import (
	"io"
)

// TemplateComponent is an interface for templ components
type TemplateComponent interface {
	Render(contentType string, w io.Writer) error
}
