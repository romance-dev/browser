/*
CODE FROM https://github.com/nearlynithin/go-ascii
*/

package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"
	// "golang.org/x/image/draw"
)

var imageCache = sync.Map{}

func isRelative(u *url.URL) bool {
	// A URL is absolute if it has both a Scheme and a Host.
	return u.Scheme == "" && u.Host == ""
}

func parseTags(htmlSource string, baseURL *url.URL, downloadImages bool, linkURLs, linkDescs *[]string, title *string) error {
	imageURLs := []string{}

	doc, err := html.Parse(strings.NewReader(htmlSource))
	if err != nil {
		return err
	}

	var visitNode func(*html.Node)
	visitNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "head" {
			for c1 := range n.ChildNodes() {
				if c1.Type == html.ElementNode && c1.Data == "title" {
					for c2 := range c1.ChildNodes() {
						if c2.Type == html.TextNode {
							*title = c2.Data
							break
						}
					}
				}
			}
		} else if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					// src could be "image1.jpg" or "/images/image2.png" or full url

					u, err := url.Parse(attr.Val)
					if err != nil {
						return
					}

					if isRelative(u) {
						u = baseURL.ResolveReference(u)
					}

					fullURL := u.String()
					imageURLs = append(imageURLs, fullURL)
				}
			}
		} else if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					*linkURLs = append(*linkURLs, attr.Val)
				}
			}

			txtData := ""
			for v := range n.Descendants() {
				if v.Type == html.TextNode {
					txtData = txtData + v.Data
				}
			}
			*linkDescs = append(*linkDescs, txtData)
		}

		// Recursively visit child and sibling nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visitNode(c)
		}
	}

	visitNode(doc)

	if downloadImages {
		var wg sync.WaitGroup
		for _, fullURL := range imageURLs {
			wg.Go(func() {
				if _, exists := imageCache.Load(fullURL); !exists {
					go func(fullURL string) {
						resp, err := http.Get(fullURL)
						if err != nil {
							return
						}
						defer resp.Body.Close()

						m, _, err := image.Decode(resp.Body)
						if err != nil {
							return
						}
						imageCache.Store(fullURL, m)
					}(fullURL)
				}
			})
		}
		wg.Wait()
	}
	return nil
}

func downloadImage(src string, terminalWidth int) (image.Image, error) {
	val, ok := imageCache.Load(src)
	if ok {
		return val.(image.Image), nil
	}

	resp, err := http.Get(src)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	m, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	return m, nil

	// // Scale image
	// b := m.Bounds()
	// fmt.Println(b)
	// ar := float64(b.Max.X-b.Min.X) / float64(b.Max.Y-b.Min.Y) // width / height
	// fmt.Println(ar)
	// newWidth := (terminalWidth)
	// newHeight := int(float64(terminalWidth) / ar)
	//
	// dst := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))
	// fmt.Println(dst.Rect)
	// draw.CatmullRom.Scale(dst, dst.Rect, m, m.Bounds(), draw.Over, nil)
	// fmt.Println(dst.Rect)
	//
	// return dst, nil
}
