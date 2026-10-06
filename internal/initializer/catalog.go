package initializer

type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	InitCommand string   `json:"initCommand"`
}

func Templates() []Template {
	return []Template{
		{
			Name: "basic", Description: "Dependency-free static web app.",
			Permissions: []string{}, InitCommand: "velox init my-app --template basic",
		},
		{
			Name: "text-editor", Description: "Native text editor with local draft recovery.",
			Permissions: []string{"file.open", "file.save"}, InitCommand: "velox init my-editor --template text-editor",
		},
		{
			Name: "folder-browser", Description: "Read-only folder listing and UTF-8 text preview.",
			Permissions: []string{"folder.read", "folder.readText"}, InitCommand: "velox init my-browser --template folder-browser",
		},
		{
			Name: "tray-app", Description: "Single-instance tray app with notifications and window state recovery.",
			Permissions: []string{"notification.show"}, InitCommand: "velox init my-tray --template tray-app",
		},
	}
}

func findTemplate(name string) (Template, bool) {
	for _, template := range Templates() {
		if template.Name == name {
			return template, true
		}
	}
	return Template{}, false
}
