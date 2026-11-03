//go:build darwin && cgo

package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// Add one source link per contents entry and a target at its heading. Quartz
// resolves these private URLs to final page numbers and internal destinations.
func pdfContents(body, filename, host string) (string, error) {
	tree, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", err
	}
	var article, title *html.Node
	var headings []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "body" {
				article = n
			}
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				headings = append(headings, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(tree)
	element := func(tag string) *html.Node { return &html.Node{Type: html.ElementNode, Data: tag} }
	text := func(value string) *html.Node { return &html.Node{Type: html.TextNode, Data: value} }
	var plain func(io.Writer, *html.Node)
	plain = func(w io.Writer, n *html.Node) {
		if n.Type == html.TextNode {
			io.WriteString(w, n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			plain(w, c)
		}
	}
	if len(headings) > 0 && headings[0].Data == "h1" && headings[0].Parent == article {
		title = headings[0]
		headings = headings[1:]
		article.RemoveChild(title)
	} else {
		title = element("h1")
		title.AppendChild(text(filename))
	}
	article.InsertBefore(title, article.FirstChild)
	if len(headings) > 0 {
		toc := element("nav")
		toc.Attr = []html.Attribute{{Key: "class", Val: "pdf-toc"}}
		h := element("h2")
		h.AppendChild(text("Contents"))
		toc.AppendChild(h)
		article.InsertBefore(toc, title.NextSibling)
		for i, n := range headings {
			var label strings.Builder
			plain(&label, n)
			row := element("div")
			row.Attr = []html.Attribute{{Key: "style", Val: fmt.Sprintf("padding-left:%dem", int(n.Data[1]-'1'))}}
			link := element("a")
			link.Attr = []html.Attribute{{Key: "href", Val: fmt.Sprintf("https://%s/entry/%d", host, i)}}
			link.AppendChild(text(label.String()))
			row.AppendChild(link)
			toc.AppendChild(row)
			marker := element("a")
			marker.Attr = []html.Attribute{{Key: "href", Val: fmt.Sprintf("https://%s/heading/%d", host, i)}, {Key: "class", Val: "pdf-target"}}
			marker.AppendChild(text("."))
			n.InsertBefore(marker, n.FirstChild)
		}
	}
	var result bytes.Buffer
	for n := article.FirstChild; n != nil; n = n.NextSibling {
		if err := html.Render(&result, n); err != nil {
			return "", err
		}
	}
	return result.String(), nil
}
