package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	nurl "net/url"
	"os"
	"strings"

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

// theme

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{}

	for _, site := range defaultMenu {
		s = append(s, site)
	}

	for _, site := range history {
		s = append(s, site)
	}

	for _, site := range siteLinks {
		s = append(s, site)
	}

	for _, site := range siteDesc {
		s = append(s, site)
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
		{Text: "-o", Description: "Open site in default browser"},
		{Text: "-f", Description: "Disable Reading Mode (show full site)"},
		{Text: "-i", Description: "Enable image rendering (experimental)"},
		{Text: "-q", Description: "Quit after rendering page"},
		{Text: "exit", Description: "Exit browser"},
		{Text: "help", Description: "Show Help"},
		{Text: "version", Description: "Display Version"},
	}
	history       = make([]prompt.Suggest, 0)
	promptHistory = make([]string, 0)
	siteLinks     = make([]prompt.Suggest, 0)
	siteDesc      = make([]prompt.Suggest, 0)
)

var jar *cookiejar.Jar

func main() {
	jar, _ = cookiejar.New(nil)
	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) > 0 && argsWithoutProg[0] == "version" {
		fmt.Printf("%s (v%v)\n", appName, appVersion)
		os.Exit(0)
	}

	if len(argsWithoutProg) > 0 && argsWithoutProg[0] == "help" {
		displayHelp()
		os.Exit(0)
	}

	appTitle()

OUTER:
	for {
		var url string
		var summary bool = true
		var downloadImages bool = false
		var quitImmediately bool = false
		if len(argsWithoutProg) > 0 {
			url = argsWithoutProg[len(argsWithoutProg)-1]
			for _, f := range argsWithoutProg {
				if f == "-o" {
					browser.OpenURL(url)
					argsWithoutProg = argsWithoutProg[:0]
					continue OUTER
				} else if f == "-f" {
					summary = false
				} else if f == "-i" {
					downloadImages = true
				} else if f == "-q" {
					quitImmediately = true
				}
			}
			argsWithoutProg = argsWithoutProg[:0]
		} else {
			url = prompt.Input("site: ", completer, prompt.OptionHistory(promptHistory), prompt.OptionMaxSuggestion(uint16(len(defaultMenu))), prompt.OptionPrefixTextColor(prompt.Red))
		}

		url = strings.TrimSpace(url)

		switch url {
		case "exit", "close", "quit":
			os.Exit(0)
		case "version":
			fmt.Println("v" + appVersion)
			continue OUTER
		case "help":
			displayHelp()
			continue OUTER
		}

		if strings.HasPrefix(url, "-o") {
			browser.OpenURL(strings.TrimSpace(strings.TrimPrefix(url, "-o")))
			continue OUTER
		} else if strings.HasPrefix(url, "-f -i") {
			summary = false
			downloadImages = true
			url = strings.TrimSpace(strings.TrimPrefix(url, "-f -i"))
		} else if strings.HasPrefix(url, "-i -f") {
			summary = false
			downloadImages = true
			url = strings.TrimSpace(strings.TrimPrefix(url, "-i -f"))
		} else if strings.HasPrefix(url, "-i -q") {
			downloadImages = true
			quitImmediately = true
			url = strings.TrimSpace(strings.TrimPrefix(url, "-i -q"))
		} else if strings.HasPrefix(url, "-q -i") {
			downloadImages = true
			quitImmediately = true
			url = strings.TrimSpace(strings.TrimPrefix(url, "-q -i"))
		} else if strings.HasPrefix(url, "-f") {
			summary = false
			url = strings.TrimSpace(strings.TrimPrefix(url, "-f"))
		} else if strings.HasPrefix(url, "-i") {
			downloadImages = true
			url = strings.TrimSpace(strings.TrimPrefix(url, "-i"))
		} else if strings.HasPrefix(url, "-q") {
			quitImmediately = true
			url = strings.TrimSpace(strings.TrimPrefix(url, "-q"))
		}

		baseURL, err := nurl.Parse(url)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}

		// Check if it's a valid url (It could be a link description instead based on how we store both)
		for _, s := range siteDesc {
			if url == s.Text {
				url = s.Description
				break
			}
		}

		siteLinks = siteLinks[:0]
		siteDesc = siteDesc[:0]

		client := &http.Client{
			Jar: jar,
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}
		req.Header.Set("User-Agent", chromeUserAgent)
		req.Header.Set("Accept", `text/html, text/html;q=0.9, */*;q=0.8`)
		req.Header.Set("Sec-CH-UA-Mobile", `?1`)

		color.Green("Loading site...")

		resp, err := client.Do(req)
		if err != nil {
			// Get "https://ninemsn.com.au": net/http: TLS handshake timeout
			color.Red("❌ Could not render site: " + err.Error())
			continue OUTER
		}
		defer resp.Body.Close()

		history = append(history, prompt.Suggest{Text: url})
		promptHistory = append(promptHistory, url)

		var htmlSource string

		pageTitle := ""
		if summary {
			// Extract readable bits only
			read := readability.New()
			article, err := read.Parse(resp.Body, url)
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

		testingHTML := `
		
		<h1>Hello</h1>
		
		<a href="http://ninemsn.com.au">Welcome</a>
		

		
		<a href="http://ninemsn.com.au">func GetState[V any](ctx context.Context, key string) V</a>
		
		<br>
		<img alt="alt image" src="https://png.pngtree.com/png-clipart/20200225/original/pngtree-image-of-cute-radish-vector-or-color-illustration-png-image_5274337.jpg" />
		
		<br/>
		<br/>
		<a href="http://ninemsn.com.au"><img alt="alt image" src="https://png.pngtree.com/png-clipart/20200225/original/pngtree-image-of-cute-radish-vector-or-color-illustration-png-image_5274337.jpg" /></a>

		`

		_ = testingHTML

		// Download all images and store in cache + record link information
		linkURLs := []string{}  // Contains the urls
		linkDescs := []string{} // Contains the url descriptions

		parseTags(htmlSource, baseURL, downloadImages, &linkURLs, &linkDescs, &pageTitle)

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

		// Convert html to markdown (https://github.com/JohannesKaufmann/html-to-markdown)
		conv := converter.NewConverter(
			converter.WithPlugins(
				base.NewBasePlugin(),
				commonmark.NewCommonmarkPlugin(),
				strikethrough.NewStrikethroughPlugin(),
				table.NewTablePlugin(),
			),
		)

		if downloadImages {
			conv.Register.RendererFor("img", converter.TagTypeInline, renderImg(twidth, downloadImages), converter.PriorityEarly)
		}

		markdown, err := conv.ConvertString(
			htmlSource,
			// testingHTML,
			converter.WithDomain(fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)),
		)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}

		clearScreen()
		addSiteInfo(url, pageTitle, twidth)

		// Render markdown
		g, _ := glamour.NewTermRenderer(
			// glamour.WithStylePath("dark"),
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(twidth),
		)

		out, err := g.Render(markdown)
		if err != nil {
			color.Red("❌ Could not render site")
			continue OUTER
		}

		fmt.Println(out)

		if quitImmediately {
			os.Exit(0)
		}
	}
}
