package initializer

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var invalidSlug = regexp.MustCompile(`[^a-z0-9-]+`)

type Result struct {
	Directory string   `json:"directory"`
	AppID     string   `json:"appId"`
	AppName   string   `json:"appName"`
	Files     []string `json:"files"`
}

type manifestFile struct {
	SchemaVersion int `json:"schemaVersion"`
	App           struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"app"`
	Assets struct {
		Root  string `json:"root"`
		Entry string `json:"entry"`
	} `json:"assets"`
	Window struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"window"`
	Security struct {
		Permissions []string `json:"permissions"`
	} `json:"security"`
}

type plannedFile struct {
	path string
	data []byte
}

func Create(directory string) (Result, error) {
	if strings.TrimSpace(directory) == "" {
		directory = "."
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return Result{}, fmt.Errorf("resolve project directory: %w", err)
	}
	base := filepath.Base(filepath.Clean(absolute))
	slug := projectSlug(base)
	if slug == "" {
		return Result{}, errors.New("project directory name must contain an ASCII letter or digit")
	}
	name := displayName(base)
	if name == "" {
		name = slug
	}
	appID := "dev.actutum." + slug
	rootExisted := pathExists(absolute)
	webExisted := pathExists(filepath.Join(absolute, "web"))

	manifest := manifestFile{SchemaVersion: 1}
	manifest.App.ID, manifest.App.Name, manifest.App.Version = appID, name, "0.1.0"
	manifest.Assets.Root, manifest.Assets.Entry = "web", "index.html"
	manifest.Window.Width, manifest.Window.Height = 960, 640
	manifest.Security.Permissions = []string{}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode project manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')

	files := []plannedFile{
		{path: "actutum.json", data: manifestData},
		{path: "web/index.html", data: []byte(indexHTML(name))},
		{path: "web/style.css", data: []byte(styleCSS)},
		{path: "web/app.js", data: []byte(appJS)},
	}
	for _, file := range files {
		_, statErr := os.Lstat(filepath.Join(absolute, filepath.FromSlash(file.path)))
		if statErr == nil {
			return Result{}, fmt.Errorf("refusing to overwrite %s", file.path)
		}
		if !os.IsNotExist(statErr) {
			return Result{}, fmt.Errorf("inspect planned path %s: %w", file.path, statErr)
		}
	}

	created := make([]string, 0, len(files))
	rollback := func() {
		for index := len(created) - 1; index >= 0; index-- {
			_ = os.Remove(created[index])
		}
		if !webExisted {
			_ = os.Remove(filepath.Join(absolute, "web"))
		}
		if !rootExisted {
			_ = os.Remove(absolute)
		}
	}
	for _, file := range files {
		fullPath := filepath.Join(absolute, filepath.FromSlash(file.path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			rollback()
			return Result{}, fmt.Errorf("create project directory: %w", err)
		}
		handle, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			rollback()
			return Result{}, fmt.Errorf("create %s: %w", file.path, err)
		}
		_, writeErr := handle.Write(file.data)
		closeErr := handle.Close()
		if writeErr != nil || closeErr != nil {
			_ = os.Remove(fullPath)
			rollback()
			return Result{}, fmt.Errorf("write %s: %w", file.path, errors.Join(writeErr, closeErr))
		}
		created = append(created, fullPath)
	}

	relative := filepath.ToSlash(directory)
	return Result{Directory: relative, AppID: appID, AppName: name, Files: []string{"actutum.json", "web/index.html", "web/style.css", "web/app.js"}}, nil
}

func projectSlug(value string) string {
	value = strings.ToLower(value)
	value = invalidSlug.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func displayName(value string) string {
	words := strings.FieldsFunc(value, func(character rune) bool {
		return character == '-' || character == '_' || unicode.IsSpace(character)
	})
	for index, word := range words {
		runes := []rune(strings.ToLower(word))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		words[index] = string(runes)
	}
	return strings.Join(words, " ")
}

func indexHTML(name string) string {
	name = html.EscapeString(name)
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'">
    <title>%s</title>
    <link rel="stylesheet" href="style.css">
    <script src="app.js" defer></script>
  </head>
  <body>
    <main>
      <p class="eyebrow">Actutum</p>
      <h1>%s</h1>
      <p>Static HTML, CSS, and JavaScript in a lightweight desktop shell.</p>
    </main>
  </body>
</html>
`, name, name)
}

const styleCSS = `:root {
  color-scheme: light dark;
  font-family: "Segoe UI", sans-serif;
  background: #111418;
  color: #f5f7fa;
}

body {
  min-height: 100vh;
  margin: 0;
  display: grid;
  place-items: center;
}

main {
  width: min(34rem, calc(100% - 3rem));
}

.eyebrow {
  color: #55d6be;
  font-size: 0.8rem;
  font-weight: 700;
  text-transform: uppercase;
}

h1 {
  margin: 0.4rem 0 0.75rem;
  font-size: 2rem;
  letter-spacing: 0;
}

p {
  line-height: 1.55;
}
`

const appJS = `document.documentElement.dataset.actutum = "ready";
`
