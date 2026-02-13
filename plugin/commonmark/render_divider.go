package commonmark

import (
	"github.com/romance-dev/browser/converter"
	"golang.org/x/net/html"
)

func (c *commonmark) renderDivider(_ converter.Context, w converter.Writer, _ *html.Node) converter.RenderStatus {
	w.WriteString("\n\n")
	w.WriteString(c.HorizontalRule)
	w.WriteString("\n\n")

	return converter.RenderSuccess
}
