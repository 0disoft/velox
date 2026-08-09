package runtimeconfig

import (
	"encoding/json"
	"reflect"
	"testing"
)

func FuzzParse(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"runtimeVersion":1,"app":{"id":"dev.velox.fuzz","name":"Fuzz","version":"1.0.0"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}}`),
		[]byte(`{"runtimeVersion":2}`),
		[]byte(`{"runtimeVersion":1,"runtimeVersion":1}`),
		[]byte(`null`),
		{0xff, 0xfe, 0xfd},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		parsed, err := Parse(data)
		if err != nil {
			return
		}
		encoded, err := json.Marshal(parsed)
		if err != nil {
			t.Fatalf("marshal accepted config: %v", err)
		}
		roundTrip, err := Parse(encoded)
		if err != nil {
			t.Fatalf("reparse accepted config: %v", err)
		}
		if !reflect.DeepEqual(parsed, roundTrip) {
			t.Fatalf("round trip changed config: %#v != %#v", parsed, roundTrip)
		}
	})
}
