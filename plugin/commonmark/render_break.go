package commonmark

import (
	"github.com/romance-dev/browser/converter"
	"golang.org/x/net/html"
)

func (c *commonmark) renderBreak(_ converter.Context, w converter.Writer, _ *html.Node) converter.RenderStatus {
	// Render a "hard line break"
	w.WriteString("  \n")
	return converter.RenderSuccess
}
