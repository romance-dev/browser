package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/fatih/color"
	"golang.org/x/net/html"
)

const (
	indent       = " "
	smallTextLen = 35 // characters (used to one-line a tag and its contents)
)

var (
	tagColor            = color.New(color.Underline, color.Bold, color.FgBlue)
	attrKeyColor        = color.New(color.Bold, color.FgMagenta)
	attrValColor        = color.New(color.FgCyan)
	commentColor        = color.New(color.Italic, color.FgGreen)
	errColor            = color.New(color.Bold, color.FgRed)
	textColor           = color.New(color.FgBlack)
	addIndents          = func(depth *int) { addNewLine(1); addIndentsNoNewLine(depth) }
	addIndentsNoNewLine = func(depth *int) {
		if *depth < 0 {
			*depth = 0
		}
		fmt.Print(strings.Repeat(indent, *depth))
	}
	addNewLine = func(n int) { fmt.Print(strings.Repeat("\n", n)) }
)

var re = regexp.MustCompile("\\s+")

func collapseWhitespace(htmlText string) string {
	// Define the regex for matching one or more whitespace characters.
	// \\s matches standard Unicode whitespace characters (space, tab, newline, etc.).

	// Replace all sequences of whitespace with a single space.
	collapsedText := re.ReplaceAllString(htmlText, " ")

	// Optionally, trim leading/trailing whitespace from the entire string.
	collapsedText = strings.TrimSpace(collapsedText)

	return collapsedText
}

// self-closing tags
func isVoid(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed",
		"hr", "img", "input", "link", "meta",
		"param", "source", "track", "wbr":
		return true
	}
	return false
}

func printHTML(r io.Reader) {
	depth := 0
	smallText := false // indicates small amount of text in text node
	openTag := false

	pre := false    // <pre> tag
	xmp := false    // <xmp> tag (Don't parse content inside)
	script := false // <script> tag
	style := false  // <style> tag

	T := html.NewTokenizer(r)

NEXT_TOKEN:
	for {
		tokenType := T.Next() // See: https://pkg.go.dev/golang.org/x/net/html#TokenType

		switch tokenType {
		case html.ErrorToken:
			err := T.Err()
			if err == io.EOF {
				break NEXT_TOKEN
			}
			errColor.Print(err.Error())
		case html.CommentToken:
			// A CommentToken looks like <!--x-->
			depth++

			comment := strings.TrimSpace(b2s(T.Text()))
			if len(comment) > 0 {
				addIndents(&depth)
				commentColor.Print("<!--")

				if strings.Contains(comment, "\n") { // comment is multi-line
					for line := range strings.Lines(comment) {
						addIndents(new(depth + 1))
						commentColor.Print(line)
					}
					addIndents(&depth)
				} else {
					commentColor.Printf(" %s ", comment)
				}
				commentColor.Print("-->")
			}

			depth--
		case html.DoctypeToken:
			// A DoctypeToken looks like <!DOCTYPE x>
			depth++
			addIndents(&depth)
			commentColor.Printf("<!DOCTYPE %s>", b2s(T.Text()))
			depth--
		case html.StartTagToken, html.SelfClosingTagToken:
			depth++

			_tagName, hasAttr := T.TagName()
			tag := b2s(_tagName)
			if !isVoid(tag) {
				switch tag {
				case "pre":
					pre = true // Don't collapse text node
				case "xmp":
					xmp = true // Don't collapse text node
				case "script":
					script = true // Display using chroma package
				case "style":
					style = true
				}
			}

			addIndents(&depth)
			tagColor.Print("<", tag)
			if hasAttr {
				for {
					key, value, more := T.TagAttr()
					attrKeyColor.Printf(" %s=\"", b2s(key))
					attrValColor.Printf("%s", b2s(value))
					attrKeyColor.Print(`"`)
					if !more {
						break
					}
				}
			}

			if isVoid(tag) {
				tagColor.Print("/>")
				openTag = false
				depth--
			} else {
				openTag = true
				tagColor.Print(">")
			}
		case html.EndTagToken:
			_tagName, _ := T.TagName()
			tag := b2s(_tagName)
			switch tag {
			case "pre":
				pre = false
			case "xmp":
				xmp = false
			case "script":
				script = false
			case "style":
				style = false
			}

			if !(openTag && smallText) {
				addIndents(&depth)
			}

			tagColor.Printf("</%s>", tag)

			openTag = false
			smallText = false
			depth--
		case html.TextToken:
			txt := b2s(T.Text())
			if script {
				for line := range strings.Lines(txt) {
					addIndentsNoNewLine(new(depth + 1))
					err := quick.Highlight(os.Stdout, line, "javascript", "terminal256", "xcode")
					if err != nil {
						errColor.Print(line)
					}
				}
			} else if style {
				for line := range strings.Lines(txt) {
					addIndentsNoNewLine(new(depth + 1))
					err := quick.Highlight(os.Stdout, line, "css", "terminal256", "xcode")
					if err != nil {
						errColor.Print(line)
					}
				}
			} else if pre || xmp {
				// Don't collapse text
				fmt.Print(txt)
			} else {
				trimmed := strings.TrimSpace(collapseWhitespace(txt))
				if len(trimmed) > 0 && len(trimmed) <= smallTextLen && !strings.Contains(trimmed, "\n") {
					smallText = true
					if !openTag {
						addIndents(new(depth + 1))
					}
					textColor.Print(trimmed)
				} else {
					smallText = false
					i := 0
					for line := range strings.Lines(trimmed) {
						if i == 0 {
							textColor.Print("\n")
						}
						addIndentsNoNewLine(new(depth + 1))
						textColor.Print(line)
						textColor.Print("\n")
						i++
					}
				}
			}
		}
	}
	addNewLine(2)
}
