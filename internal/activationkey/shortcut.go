package activationkey

import (
	"encoding/json"
	"errors"
	"strings"
)

type Shortcut string

type Binding struct {
	Modifiers, Key uint32
}

func (s *Shortcut) UnmarshalJSON(raw []byte) error {
	var value string
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &value) != nil {
		return errors.New("window.activationShortcut must be a string")
	}
	*s = Shortcut(value)
	return nil
}

func Parse(shortcut Shortcut) (Binding, error) {
	if shortcut == "" {
		return Binding{}, nil
	}
	parts := strings.Split(string(shortcut), "+")
	if len(parts) < 3 || len(parts) > 4 || parts[0] != "Ctrl" || parts[1] != "Alt" || (len(parts) == 4 && parts[2] != "Shift") {
		return Binding{}, invalid()
	}
	key := parts[len(parts)-1]
	if len(key) != 1 || !((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= '0' && key[0] <= '9')) {
		return Binding{}, invalid()
	}
	binding := Binding{Modifiers: 3, Key: uint32(key[0])} // MOD_CONTROL | MOD_ALT.
	if len(parts) == 4 {
		binding.Modifiers |= 4 // MOD_SHIFT.
	}
	return binding, nil
}

func invalid() error {
	return errors.New("window.activationShortcut requires Ctrl+Alt+[Shift+] and one uppercase A-Z or digit 0-9")
}
