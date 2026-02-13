package main

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/JohannesKaufmann/dom"
	"github.com/fatih/color"
	"github.com/qeesung/image2ascii/convert"
	"github.com/romance-dev/browser/converter"
	"golang.org/x/net/html"
)

// Transform <img>
func renderImg(terminalWidth int, downloadImages bool) converter.HandleRenderFunc {
	return func(ctx converter.Context, w converter.Writer, node *html.Node) converter.RenderStatus {
		/*
			<img alt="alt text" src="/image.png"> => ![alt text](/image.png)
		*/

		if src, ok := dom.GetAttribute(node, "src"); ok {
			imageData, err := downloadImage(src, terminalWidth)
			if err != nil {
				if alt, ok := dom.GetAttribute(node, "alt"); ok {
					w.WriteString(fmt.Sprintf("[Image: %s]", alt))
				}
				return converter.RenderSuccess
			}

			var sb strings.Builder
			sb.WriteString("\n```\n")

			convertOptions := convert.DefaultOptions
			converter := convert.NewImageConverter()
			convertOptions.FitScreen = true
			convertOptions.Colored = true
			convertOptions.Reversed = true
			sb.WriteString(converter.Image2ASCIIString(imageData, &convertOptions))

			if alt, ok := dom.GetAttribute(node, "alt"); ok && alt != "" {
				fmt.Fprint(&sb, "\n")
				color.New(color.Underline, color.FgBlack, color.Bold).Fprintln(&sb, alt)
			}

			sb.WriteString("```\n")

			// Display image instead
			w.WriteString(sb.String())

		}

		return converter.RenderSuccess
	}
}
