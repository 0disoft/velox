package singleinstance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func identityKey(sid string, session uint32, appID, profile string) string {
	// Encode fields separately so separators in a path cannot alias another identity.
	data, _ := json.Marshal(struct {
		SID            string
		Session        uint32
		AppID, Profile string
	}{sid, session, appID, profile})
	return fmt.Sprintf("Velox.SingleInstance.%x", sha256.Sum256(data))
}
