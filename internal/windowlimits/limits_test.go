package windowlimits

import "testing"

func TestValidateMinimums(t *testing.T) {
	for _, values := range [][4]uint{{960, 640, 0, 0}, {960, 640, 640, 480}, {960, 640, 0, 640}, {960, 640, 960, 0}, {16384, 16384, 16384, 16384}} {
		if err := Validate(values[0], values[1], values[2], values[3]); err != nil {
			t.Fatal(values, err)
		}
	}
	for _, values := range [][4]uint{{960, 640, 961, 0}, {960, 640, 0, 641}, {20000, 20000, 16385, 0}, {20000, 20000, 0, 16385}} {
		if err := Validate(values[0], values[1], values[2], values[3]); err == nil {
			t.Fatal("accepted invalid minimums", values)
		}
	}
}
