package linux

import (
	"fmt"
)

// CreateUser creates a Linux FTP user.
func CreateUser(username, homeDir string) error {

	_, err := Execute(
		"useradd",
		"-m",
		"-d", homeDir,
		"-s", "/usr/sbin/nologin",
		username,
	)

	if err != nil {
		return fmt.Errorf("failed to create linux user: %w", err)
	}

	return nil
}
