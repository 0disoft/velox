package appidentity

import (
	"errors"
	"regexp"
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$`)

func Validate(id string) error {
	if !idPattern.MatchString(id) {
		return errors.New("app.id must be lowercase reverse-domain ASCII")
	}
	return nil
}
