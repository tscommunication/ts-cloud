package linux

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var renameUserExecute = Execute

// RenameUser renames a Linux FTP user and moves its home directory.
//
// Equivalent command:
//
//	usermod -l NEW -d NEW_HOME -m OLD
//
// Validation is intentionally strict because PPPoE usernames ultimately
// control Linux account names and FTP home paths.
func RenameUser(
	oldUsername string,
	newUsername string,
	oldHome string,
	newHome string,
) error {
	oldUsername = strings.TrimSpace(oldUsername)
	newUsername = strings.TrimSpace(newUsername)
	oldHome = filepath.Clean(strings.TrimSpace(oldHome))
	newHome = filepath.Clean(strings.TrimSpace(newHome))

	if oldUsername == "" || newUsername == "" {
		return errors.New("old and new usernames are required")
	}

	if oldUsername == newUsername {
		return nil
	}

	if oldHome == "." || oldHome == "/" ||
		newHome == "." || newHome == "/" {
		return errors.New("old and new home directories are required")
	}

	if filepath.Dir(oldHome) != filepath.Dir(newHome) {
		return errors.New(
			"FTP user rename must stay within the same root directory",
		)
	}

	if filepath.Base(oldHome) != oldUsername {
		return errors.New(
			"old FTP home directory does not match old username",
		)
	}

	if filepath.Base(newHome) != newUsername {
		return errors.New(
			"new FTP home directory does not match new username",
		)
	}

	if UserExists(newUsername) {
		return fmt.Errorf(
			"linux user %q already exists",
			newUsername,
		)
	}

	_, err := renameUserExecute(
		"sudo",
		"/usr/sbin/usermod",
		"-l",
		newUsername,
		"-d",
		newHome,
		"-m",
		oldUsername,
	)
	if err != nil {
		return fmt.Errorf("failed to rename linux user: %w", err)
	}

	return nil
}
