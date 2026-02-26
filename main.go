package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	nurl "net/url"
	"os"
	"slices"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/c-bata/go-prompt"
	"github.com/charmbracelet/glamour"
	"github.com/cixtor/readability"
	"github.com/fatih/color"
	"github.com/pkg/browser"

	"github.com/romance-dev/browser/converter"
	"github.com/romance-dev/browser/plugin/base"
	"github.com/romance-dev/browser/plugin/commonmark"
	"github.com/romance-dev/browser/plugin/strikethrough"
	"github.com/romance-dev/browser/plugin/table"
)

const appVersion = "1.0.0-alpha.5" // without v prefix

func completer(d prompt.Document) []prompt.Suggest {
	uniq := map[prompt.Suggest]struct{}{}
	s := []prompt.Suggest{}

	for _, site := range defaultMenu {
		s = append(s, site)
	}

	for _, site := range history {
		uniq[site] = struct{}{}
	}

	for _, site := range siteLinks {
		uniq[site] = struct{}{}
	}

	for _, site := range siteDesc {
		uniq[site] = struct{}{}
	}

	for k := range uniq {
		s = append(s, k)
	}

	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

const chromeUserAgent = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Mobile Safari/537.36"

var (
	defaultMenu = []prompt.Suggest{
		{Text: "http://", Description: ""},
		{Text: "https://", Description: ""},
		{Text: "http://www.", Description: ""},
		{Text: "https://www.", Description: ""},
		{Text: "-f", Description: "Disable Reading Mode (show full site)"},
		{Text: "--html", Description: "Display Page Source"},
		{Text: "-i", Description: "Enable image rendering (experimental)"},
		{Text: "-o", Description: "Open site in default browser"},
		{Text: "-q", Description: "Quit after rendering page"},
		{Text: "back", Description: "Previous page"},
		{Text: "help", Description: "Show Help"},
		{Text: "version", Description: "Display Version"},
		{Text: "exit", Description: "Exit browser"},
	}
	history            = make([]prompt.Suggest, 0)
	instructionHistory = make([]instruction, 0)
	promptHistory      = make([]string, 0)
	siteLinks          = make([]prompt.Suggest, 0)
	siteDesc           = make([]prompt.Suggest, 0)
)

type instruction struct {
	url             string
	fullsite        bool
	imageRender     bool
	quitImmediately bool
	renderSource    bool
	command         string
}

var count int

func main() {
	argsWithoutProg := os.Args[1:]

	if found, _ := checkCommand(argsWithoutProg, "version"); found {
		fmt.Printf("%s (v%v)\n", appName, appVersion)
		os.Exit(0)
	}

	if found, _ := checkCommand(argsWithoutProg, "help"); found {
		displayHelp()
		os.Exit(0)
	}

	appTitle()

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piping in html content
		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			color.Red("❌ Could not render standard input: " + err.Error())
		} else {
			lexer := lexers.Analyse(b2s(contents))
			if lexer != nil && lexer.Config().Name != "HTML" && lexer.Config().Name != "GDScript3" {
				err := quick.Highlight(os.Stdout, b2s(contents), lexer.Config().Name, "terminal256", "xcode")
				if err != nil {
					color.Red("❌ Could not render standard input: " + err.Error())
				} else {
					addNewLine(2)
				}
			} else {
				printHTML(bytes.NewReader(contents))
			}
		}

		if found, _ := checkCommand(argsWithoutProg, "-q"); found {
			os.Exit(0)
		}
	}

OUTER:
	for {
		var cmd []string
		if len(argsWithoutProg) == 0 {
			input := strings.TrimSpace(prompt.Input(fmt.Sprintf("[ref@%d] site: ", count), completer, prompt.OptionTitle(appName), prompt.OptionHistory(promptHistory), prompt.OptionMaxSuggestion(uint16(len(defaultMenu))), prompt.OptionPrefixTextColor(prompt.Red)))
			count++
			flags := []string{"-f", "--html", "-i", "-o", "-q"}
			putback := []string{}
			for _, f := range flags {
				if strings.Contains(input, " "+f) {
					putback = append(putback, f)
					input = strings.ReplaceAll(input, " "+f, "")
				} else if strings.Contains(input, f+" ") {
					putback = append(putback, f)
					input = strings.ReplaceAll(input, f+" ", "")
				}
			}
			input = strings.TrimSpace(input)

			// Check if it's a valid url (It could be a link description instead based on how we store both)
			// i.e. convert description back to url
			for _, s := range siteDesc {
				if input == s.Text {
					input = s.Description
					break
				}
			}

			input = strings.TrimSpace(input + " " + strings.Join(putback, " "))
			cmd = strings.Fields(input)
		}
		cmd = slices.Concat(cmd, argsWithoutProg)
		argsWithoutProg = argsWithoutProg[:0]
		siteLinks = siteLinks[:0]
		siteDesc = siteDesc[:0]

		if found, url := checkCommand(cmd, "-o"); found {
			if url != "" {
				browser.OpenURL(url)
			}
			continue OUTER
		}

		// Parse command into instruction
		inst := instruction{}

		if found, _ := checkCommand(cmd, "-f"); found {
			inst.fullsite = true
		}

		if found, _ := checkCommand(cmd, "-i"); found {
			inst.imageRender = true
		}

		if found, _ := checkCommand(cmd, "-q"); found {
			inst.quitImmediately = true
		}

		if found, _ := checkCommand(cmd, "--html"); found {
			inst.renderSource = true
		}

		for _, s := range cmd {
			if strings.HasPrefix(s, "-") {
				continue
			}

			if isValidURL(s) {
				inst.url = s
				break
			} else {
				inst.command = s
				break
			}
		}

		instructionHistory = append(instructionHistory, inst)

		// Process command
		switch inst.command {
		case "exit", "close", "quit":
			os.Exit(0)
		case "version":
			fmt.Println("v" + appVersion)
			continue OUTER
		case "help":
			displayHelp()
			continue OUTER
		case "back", "goback":
			// Remove "back" from instructionHistory
			instructionHistory = instructionHistory[:len(instructionHistory)-1]

			if len(instructionHistory) == 0 {
				continue OUTER
			}

			// Remove currently displayed page from instructionHistory
			instructionHistory = instructionHistory[:len(instructionHistory)-1]

			if len(instructionHistory) == 0 {
				continue OUTER
			}

			inst = instructionHistory[len(instructionHistory)-1]
		}

		baseURL, err := nurl.Parse(inst.url)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}

		client := &http.Client{Jar: jar}
		req, err := http.NewRequest("GET", inst.url, nil)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}
		req.Header.Set("User-Agent", chromeUserAgent)
		req.Header.Set("Accept", `text/html, */*;q=0.8`)
		req.Header.Set("Sec-CH-UA-Mobile", `?1`)

		color.Green("Loading site...")

		resp, err := client.Do(req)
		if err != nil {
			// Get "https://ninemsn.com.au": net/http: TLS handshake timeout
			color.Red("❌ Could not render site: " + err.Error())
			continue OUTER
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		if before, _, found := strings.Cut(contentType, ";"); found {
			contentType = before
		}

		promptHistory = append(promptHistory, strings.Join(cmd, " ")) // prompt history via up/down keys

		var htmlSource string

		pageTitle := ""
		if !inst.fullsite && contentType == "text/html" {
			// Extract readable bits only
			read := readability.New()
			article, err := read.Parse(resp.Body, inst.url)
			if err != nil {
				color.Red("❌ Could not render site")
				continue OUTER
			}
			htmlSource = article.Content
			pageTitle = article.Title
		} else {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				color.Red("❌ Could not render site")
				continue OUTER
			}

			htmlSource = b2s(bodyBytes)
		}

		twidth := terminalWidth()

		// Download all images and store in cache + record link information
		linkURLs := []string{}  // Contains the urls
		linkDescs := []string{} // Contains the url descriptions

		parseTags(htmlSource, baseURL, inst.imageRender, &linkURLs, &linkDescs, &pageTitle)

		history = append(history, prompt.Suggest{Text: inst.url, Description: pageTitle})
		if pageTitle != "" {
			history = append(history, prompt.Suggest{Text: pageTitle, Description: inst.url})
		}

		// Look for links
		for i, link := range linkURLs {
			if strings.HasPrefix(link, "#") {
				continue
			}
			desc := linkDescs[i]
			if len(link) > 0 {
				siteLinks = append(siteLinks, prompt.Suggest{Text: link, Description: desc})
			}
			if len(desc) > 0 {
				siteDesc = append(siteDesc, prompt.Suggest{Text: desc, Description: link})
			}
		}

		if inst.renderSource {
			lexer := lexers.MatchMimeType(contentType)
			if lexer == nil {
				lexer = lexers.Match(inst.url)
				if lexer == nil {
					lexer = lexers.Analyse(htmlSource)
				}
			}

			clearScreen()
			addSiteInfo(inst.url+" (source)", pageTitle, twidth)

			if lexer != nil && lexer.Config().Name != "HTML" && lexer.Config().Name != "GDScript3" {
				err := quick.Highlight(os.Stdout, htmlSource, lexer.Config().Name, "terminal256", "xcode")
				if err != nil {
					color.Red("❌ Could not display page source: " + err.Error())
				} else {
					addNewLine(2)
				}
			} else {
				printHTML(bytes.NewReader(s2b(htmlSource)))
			}
			continue OUTER
		}

		// Convert html to markdown
		conv := converter.NewConverter(
			converter.WithPlugins(
				base.NewBasePlugin(),
				commonmark.NewCommonmarkPlugin(),
				strikethrough.NewStrikethroughPlugin(),
				table.NewTablePlugin(),
			),
		)

		if inst.imageRender {
			conv.Register.RendererFor("img", converter.TagTypeInline, renderImg(twidth, inst.imageRender), converter.PriorityEarly)
		}

		markdown, err := conv.ConvertString(
			htmlSource,
			converter.WithDomain(fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)),
		)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}

		clearScreen()
		addSiteInfo(inst.url, pageTitle, twidth)

		// Render markdown
		g, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(twidth),
		)

		out, err := g.Render(markdown)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}

		fmt.Println(out)

		if inst.quitImmediately {
			os.Exit(0)
		}
	}
}
