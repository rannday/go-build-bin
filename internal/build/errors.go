package build

import (
	"errors"
	"regexp"
)

var ErrHelp = errors.New("help requested")

var versionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

func ValidateVersion(version string) error {
	if versionPattern.MatchString(version) {
		return nil
	}
	return errors.New("version must be non-empty and contain only letters, numbers, dots, underscores, plus, or hyphen")
}
