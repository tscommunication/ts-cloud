package linux

import (
	"fmt"
	"os/user"
)

// CreateUser creates a Linux FTP user.
func CreateUser(username, homeDir string) error {

	_, err := Execute(
		"sudo",
		"/usr/sbin/useradd",
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

func UserExists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}
