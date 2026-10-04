package activationkey

import (
	"encoding/json"
	"testing"
)

func TestShortcutGrammar(t *testing.T) {
	for _, test := range []struct {
		input Shortcut
		want  Binding
	}{{"", Binding{}}, {"Ctrl+Alt+V", Binding{3, 'V'}}, {"Ctrl+Alt+Shift+9", Binding{7, '9'}}} {
		got, err := Parse(test.input)
		if err != nil || got != test.want {
			t.Fatal(test.input, got, err)
		}
	}
	for _, input := range []Shortcut{" ", "Ctrl+V", "Alt+Ctrl+V", "Ctrl+Alt+v", "Ctrl+Alt+VV", "Ctrl+Alt+F12", "Win+Alt+V", "Ctrl+Alt+", "Ctrl+Alt+Shift+", "Ctrl+Alt+Alt+V", "Ctrl+Alt+Shift+Shift+V", "Ctrl+Alt+\x00", "Ctrl+Alt+\uD55C", "Ctrl+Alt+V "} {
		if _, err := Parse(input); err == nil {
			t.Fatal("accepted", input)
		}
	}
}

func TestShortcutJSONRequiresString(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `1`, `[]`, `{}`} {
		var shortcut Shortcut
		if json.Unmarshal([]byte(raw), &shortcut) == nil {
			t.Fatal("accepted", raw)
		}
	}
	var shortcut Shortcut
	if json.Unmarshal([]byte(`"Ctrl+Alt+V"`), &shortcut) != nil || shortcut != "Ctrl+Alt+V" {
		t.Fatal(shortcut)
	}
}
