package initializer

import (
	"embed"
	"fmt"
	"html"
	"strings"
)

//go:embed text-editor/* folder-browser/* tray-app/*
var templateAssets embed.FS

func templateFiles(template, name string) ([]plannedFile, error) {
	if template == "basic" {
		return []plannedFile{
			{path: "web/index.html", data: []byte(indexHTML(name))},
			{path: "web/style.css", data: []byte(styleCSS)},
			{path: "web/app.js", data: []byte(appJS)},
		}, nil
	}
	names := []string{"index.html", "style.css", "app.js", "drafts.js", "find.js", "file-plus.svg", "folder-open.svg", "save.svg", "save-all.svg", "search.svg", "chevron-up.svg", "chevron-down.svg", "x.svg", "replace.svg", "replace-all.svg", "undo-2.svg", "icons-license.txt"}
	if template == "folder-browser" {
		names = []string{"index.html", "style.css", "app.js", "folder-open.svg", "refresh-cw.svg", "x.svg", "icons-license.txt"}
	} else if template == "tray-app" {
		names = []string{"index.html", "style.css", "app.js", "bell.svg", "icons-license.txt"}
	}
	files := make([]plannedFile, 0, len(names))
	for _, filename := range names {
		data, err := templateAssets.ReadFile(template + "/" + filename)
		if err != nil {
			return nil, fmt.Errorf("read embedded template %s: %w", filename, err)
		}
		if filename == "index.html" {
			data = []byte(strings.ReplaceAll(string(data), "{{APP_NAME}}", html.EscapeString(name)))
		}
		files = append(files, plannedFile{path: "web/" + filename, data: data})
	}
	return files, nil
}
