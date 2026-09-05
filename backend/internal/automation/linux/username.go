package linux

import (
	"errors"
	"regexp"
	"strings"
)

const MaxManagedUsernameLength = 32

var managedUsernamePattern = regexp.MustCompile(
	`^[A-Za-z0-9][A-Za-z0-9_.@-]*$`,
)

// ValidateManagedUsername validates an application-managed Linux account name.
//
// TS-Cloud intentionally uses a stricter rule than useradd itself because
// useradd may accept names containing path separators or traversal-like text
// on some systems.
//
// Existing ISP PPPoE identities are preserved exactly. Uppercase letters,
// underscores and hyphens are supported; usernames are never lowercased or
// otherwise transformed.
func ValidateManagedUsername(username string) error {
	username = strings.TrimSpace(username)

	if username == "" {
		return errors.New("username is required")
	}

	if len(username) > MaxManagedUsernameLength {
		return errors.New("username exceeds 32 characters")
	}

	if !managedUsernamePattern.MatchString(username) {
		return errors.New(
			"username contains characters that are unsafe for a managed Linux account",
		)
	}

	return nil
}
