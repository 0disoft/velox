package singleinstance

import "testing"

func TestIdentitySeparatesUserSessionAppAndProfile(t *testing.T) {
	base := identityKey("S-1-5-21", 1, "dev.velox.test", `c:\profile`)
	for _, got := range []string{
		identityKey("S-1-5-22", 1, "dev.velox.test", `c:\profile`),
		identityKey("S-1-5-21", 2, "dev.velox.test", `c:\profile`),
		identityKey("S-1-5-21", 1, "dev.velox.other", `c:\profile`),
		identityKey("S-1-5-21", 1, "dev.velox.test", `c:\other`),
	} {
		if base == got {
			t.Fatal("different identity collided")
		}
	}
	if base != identityKey("S-1-5-21", 1, "dev.velox.test", `c:\profile`) {
		t.Fatal("identity is not stable")
	}
}
