package commonmark

import (
	"bytes"

	"github.com/romance-dev/browser/converter"
	"github.com/romance-dev/browser/internal/textutils"
	"golang.org/x/net/html"
)

func (c *commonmark) renderBlockquote(ctx converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
	var buf bytes.Buffer
	ctx.RenderChildNodes(ctx, &buf, n)

	content := buf.Bytes()
	content = bytes.TrimSpace(content)
	if content == nil {
		return converter.RenderSuccess
	}

	content = textutils.TrimConsecutiveNewlines(content)
	content = textutils.TrimUnnecessaryHardLineBreaks(content)
	content = textutils.PrefixLines(content, []byte{'>', ' '})

	w.WriteRune('\n')
	w.WriteRune('\n')
	w.Write(content)
	w.WriteRune('\n')
	w.WriteRune('\n')

	return converter.RenderSuccess
}
