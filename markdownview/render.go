// Package markdownview renders local Markdown and owns viewer navigation state.
// It has no desktop or cgo dependency.
package markdownview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"go.yaml.in/yaml/v3"
	nethtml "golang.org/x/net/html"
)

type Heading struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Level int    `json:"level"`
}
type Document struct {
	HTML     string    `json:"html"`
	Headings []Heading `json:"headings"`
	Warnings []string  `json:"warnings"`
}

const MaxDocument = 8 << 20
const MaxImage = 20 << 20

func ReadFile(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("file exceeds %d MiB: %s", limit>>20, path)
	}
	return b, err
}

func Render(source []byte, path string) (Document, error) {
	d := Document{Headings: []Heading{}, Warnings: []string{}}
	if len(source) > MaxDocument {
		return d, fmt.Errorf("Markdown exceeds 8 MiB")
	}
	source = bytes.TrimPrefix(source, []byte{0xef, 0xbb, 0xbf})
	source = bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	front, body, warning := frontmatter(source)
	if warning != "" {
		d.Warnings = append(d.Warnings, warning)
	}
	var out bytes.Buffer
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote), goldmark.WithRendererOptions(gmhtml.WithUnsafe()))
	if err := md.Convert(body, &out); err != nil {
		return d, err
	}
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	policy.AllowAttrs("disabled", "checked").OnElements("input")
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^frontmatter$`)).OnElements("table")
	policy.AllowAttrs("id").OnElements("a")
	// URLs are classified below, before any markup reaches the privileged webview.
	policy.AllowURLSchemes("file")
	policy.AllowDataURIImages()
	tree, err := nethtml.Parse(strings.NewReader(policy.Sanitize(front + out.String())))
	if err != nil {
		return d, err
	}
	used := map[string]bool{}
	imageBytes := 0
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			if n.Data == "input" {
				checked := false
				for _, a := range n.Attr {
					if a.Key == "checked" {
						checked = true
					}
				}
				n.Attr = []nethtml.Attribute{{Key: "type", Val: "checkbox"}, {Key: "disabled", Val: ""}}
				if checked {
					setAttr(n, "checked", "")
				}
			}
			if n.Data == "li" || n.Data == "h1" || n.Data == "h2" || n.Data == "h3" || n.Data == "h4" || n.Data == "h5" || n.Data == "h6" {
				decorateTask(n)
			}
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				title := nodeText(n)
				base := slug(title)
				id := base
				for i := 1; used[id]; i++ {
					id = fmt.Sprintf("%s-%d", base, i)
				}
				used[id] = true
				setAttr(n, "id", id)
				d.Headings = append(d.Headings, Heading{id, title, int(n.Data[1] - '0')})
			}
			if n.Data == "img" {
				src := attr(n, "src")
				u, e := url.Parse(src)
				if e == nil && (u.Scheme == "" || u.Scheme == "file") && u.Host == "" {
					p := u.Path
					if !filepath.IsAbs(p) {
						p = filepath.Join(filepath.Dir(path), p)
					}
					b, e := ReadFile(p, MaxImage)
					if e == nil {
						mime := http.DetectContentType(b)
						if strings.EqualFold(filepath.Ext(p), ".svg") {
							mime = "image/svg+xml"
						}
						if !strings.HasPrefix(mime, "image/") {
							e = fmt.Errorf("not an image: %s", p)
						} else if imageBytes+len(b) > 40<<20 {
							e = fmt.Errorf("local images exceed 40 MiB")
						} else {
							imageBytes += len(b)
							setAttr(n, "src", "data:"+mime+";base64,"+base64.StdEncoding.EncodeToString(b))
						}
					}
					if e != nil {
						setAttr(n, "src", "")
						d.Warnings = append(d.Warnings, e.Error())
					}
				} else if e != nil || !(u.Scheme == "https" || u.Scheme == "http" || u.Scheme == "data") {
					setAttr(n, "src", "")
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(tree)
	var findBody func(*nethtml.Node) *nethtml.Node
	findBody = func(n *nethtml.Node) *nethtml.Node {
		if n.Data == "body" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if b := findBody(c); b != nil {
				return b
			}
		}
		return nil
	}
	out.Reset()
	for c := findBody(tree).FirstChild; c != nil; c = c.NextSibling {
		nethtml.Render(&out, c)
	}
	d.HTML = out.String()
	return d, nil
}

func attr(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func setAttr(n *nethtml.Node, key, value string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = value
			return
		}
	}
	n.Attr = append(n.Attr, nethtml.Attribute{Key: key, Val: value})
}
func nodeText(n *nethtml.Node) string {
	if n.Type == nethtml.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(nodeText(c))
	}
	return b.String()
}
func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "section"
	}
	return b.String()
}
func decorateTask(n *nethtml.Node) {
	p := n
	if p.FirstChild != nil && p.FirstChild.Data == "p" {
		p = p.FirstChild
	}
	c := p.FirstChild
	if c == nil || c.Type != nethtml.TextNode || len(c.Data) < 3 {
		return
	}
	marker := c.Data[:3]
	if marker != "[>]" && marker != "[ ]" && marker != "[x]" && marker != "[X]" {
		return
	}
	if len(c.Data) > 3 && c.Data[3] != ' ' && c.Data[3] != '\n' {
		return
	}
	c.Data = strings.TrimLeft(c.Data[3:], " ")
	state, label, glyph := "pending", "Pending", "☐"
	if marker == "[>]" {
		state, label, glyph = "running", "In progress", "▶"
	} else if marker != "[ ]" {
		state, label, glyph = "done", "Completed", "☑"
	}
	span := &nethtml.Node{Type: nethtml.ElementNode, Data: "span", Attr: []nethtml.Attribute{{Key: "class", Val: "task-status " + state}, {Key: "aria-label", Val: label}, {Key: "title", Val: label}}}
	span.AppendChild(&nethtml.Node{Type: nethtml.TextNode, Data: glyph + " "})
	p.InsertBefore(span, c)
}
func frontmatter(source []byte) (string, []byte, string) {
	if !bytes.HasPrefix(source, []byte("---\n")) {
		return "", source, ""
	}
	end := -1
	offset := 4
	for _, line := range bytes.Split(source[4:], []byte("\n")) {
		if string(line) == "---" || string(line) == "..." {
			end = offset
			break
		}
		offset += len(line) + 1
	}
	if end < 0 {
		return "", source, "Unclosed YAML frontmatter"
	}
	var root yaml.Node
	if err := yaml.Unmarshal(source[4:end], &root); err != nil {
		return "", source, "Invalid YAML frontmatter: " + err.Error()
	}
	if len(root.Content) == 0 {
		return "", source[min(end+4, len(source)):], ""
	}
	m := root.Content[0]
	if m.Kind != yaml.MappingNode {
		return "", source, "Frontmatter must be a YAML mapping"
	}
	var b strings.Builder
	b.WriteString(`<table class="frontmatter"><tbody>`)
	for i := 0; i < len(m.Content); i += 2 {
		v := m.Content[i+1]
		value := v.Value
		if v.Kind != yaml.ScalarNode {
			raw, e := yaml.Marshal(v)
			if e != nil {
				return "", source, e.Error()
			}
			value = strings.TrimSpace(string(raw))
		}
		fmt.Fprintf(&b, "<tr><th>%s</th><td><pre>%s</pre></td></tr>", html.EscapeString(m.Content[i].Value), html.EscapeString(value))
	}
	b.WriteString("</tbody></table>")
	return b.String(), source[min(end+4, len(source)):], ""
}
