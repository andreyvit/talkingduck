//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework Cocoa -framework WebKit -framework PDFKit
#include <stdlib.h>
int tdWritePDF(const char *html, const char *path, const char *host, int dark);
*/
import "C"
import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/zkrebbekx/go-mermaid"
	"talkingduck/markdownview"
	"talkingduck/settings"
)

func exportPDF(path string, doc markdownview.Document, appearance settings.MarkdownAppearance, printable bool) (string, error) {
	css, err := frontend.ReadFile("frontend/github-markdown-dark.css")
	if err != nil {
		return "", err
	}
	layout, err := frontend.ReadFile("frontend/export.css")
	if err != nil {
		return "", err
	}
	body := doc.HTML
	class, suffix, dark := "ebook", "", C.int(1)
	if printable {
		class, suffix, dark = "printable", "-print", 0
		for _, image := range doc.Images {
			if image.Mermaid == "" {
				continue
			}
			svg, err := mermaid.Render(image.Mermaid, mermaid.WithTheme(mermaid.Neutral))
			if err != nil {
				return "", err
			}
			body = strings.Replace(body, image.Src, "data:image/svg+xml;base64,"+base64.StdEncoding.EncodeToString(svg), 1)
		}
	}
	host := fmt.Sprintf("%x.invalid", rand.Text())
	body, err = pdfContents(body, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), host)
	if err != nil {
		return "", err
	}
	a := appearance.Defaults()
	font := "-apple-system, sans-serif"
	if a.Font != "github" {
		quoted, _ := json.Marshal(a.Font)
		font = string(quoted) + ", sans-serif"
	}
	page := fmt.Sprintf(`<!doctype html><html class="%s"><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data: https: http:; style-src 'unsafe-inline'"><title>%s</title><style>%s%s:root{--font:%s;--size:13px}</style><body><article class="markdown-body">%s</article></body></html>`, class, html.EscapeString(filepath.Base(path)), css, layout, font, body)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".talkingduck-export-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Close(); err != nil {
		return "", err
	}
	content, dest := C.CString(page), C.CString(tmp.Name())
	defer C.free(unsafe.Pointer(content))
	defer C.free(unsafe.Pointer(dest))
	markerHost := C.CString(host)
	defer C.free(unsafe.Pointer(markerHost))
	if C.tdWritePDF(content, dest, markerHost, dark) == 0 {
		return "", fmt.Errorf("WebKit could not export %s", filepath.Base(path))
	}
	base := strings.TrimSuffix(path, filepath.Ext(path)) + suffix
	for n := 1; ; n++ {
		name := base + ".pdf"
		if n > 1 {
			name = fmt.Sprintf("%s (%d).pdf", base, n)
		}
		if err = os.Link(tmp.Name(), name); os.IsExist(err) {
			continue
		}
		return name, err
	}
}

func (v *markdownViewer) export(path string, doc markdownview.Document, printable bool) {
	data, err := v.store.Load()
	if err == nil {
		var dest string
		dest, err = exportPDF(path, doc, data.MarkdownAppearance, printable)
		if err == nil {
			err = exec.Command("/usr/bin/open", "-R", dest).Run()
		}
	}
	v.mu.Lock()
	v.exporting = false
	v.mu.Unlock()
	if err != nil {
		v.app.Dialog.Error().SetTitle("PDF export").SetMessage(err.Error()).Show()
	}
}
