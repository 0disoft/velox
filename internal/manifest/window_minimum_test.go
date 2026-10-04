package manifest

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestLoadMinimumWindowSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "velox.json")
	for _, test := range []struct {
		fields        string
		width, height uint
		valid         bool
	}{
		{`"width":960,"height":640`, 0, 0, true},
		{`"minWidth":640,"minHeight":480`, 640, 480, true},
		{`"minWidth":640`, 640, 0, true},
		{`"minWidth":0,"minHeight":0`, 0, 0, true},
		{`"minWidth":961`, 0, 0, false}, {`"minHeight":641`, 0, 0, false},
		{`"width":20000,"minWidth":16385`, 0, 0, false},
		{`"minWidth":-1`, 0, 0, false}, {`"minHeight":1.5`, 0, 0, false},
	} {
		writeTestFile(t, path, fmt.Sprintf(`{"schemaVersion":1,"app":{"id":"dev.velox.minimum","name":"Minimum","version":"1"},"window":{%s}}`, test.fields))
		got, err := Load(path)
		if (err == nil) != test.valid {
			t.Fatalf("%s: %v", test.fields, err)
		}
		if test.valid && (got.Window.MinWidth != test.width || got.Window.MinHeight != test.height) {
			t.Fatal(got.Window)
		}
	}
}
