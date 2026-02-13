package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/rivo/uniseg"
	"golang.org/x/term"
)

const (
	appName    = "Terminal Browser"
	appVersion = "1.0.0-alpha" // without v prefix
)

func appTitle() {
	title := fmt.Sprintf("%s (v%v)", appName, appVersion)
	titleWidth := uniseg.StringWidth(title)
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		width, _, _ := term.GetSize(fd)
		if width == 0 {
			color.New(color.BgHiRed, color.FgWhite, color.Bold).Println(title)
		} else {
			margin := int((width - titleWidth) / 2)
			fmt.Print(strings.Repeat(" ", margin))
			color.New(color.BgHiRed, color.FgWhite, color.Bold).Print(title)
			fmt.Println(strings.Repeat(" ", margin))
		}
	}
}

func addSiteInfo(url string, title string, twidth int) {
	maxWidth := 0
	d := color.New(color.FgRed, color.Bold)
	d.Printf("URL: ")
	d = color.New(color.FgRed)
	d.Println(url)
	maxWidth = uniseg.StringWidth("URL: " + url)
	if title != "" {
		d = color.New(color.FgRed, color.Bold)
		d.Printf("Title: ")
		d = color.New(color.FgRed)
		d.Println(title)
		if titleWidth := uniseg.StringWidth("Title: " + title); titleWidth > maxWidth {
			maxWidth = titleWidth
		}
	}

	if twidth != 0 && maxWidth > twidth {
		maxWidth = twidth
	}

	d = color.New(color.FgBlue)
	d.Println(strings.Repeat("=", maxWidth))
}
