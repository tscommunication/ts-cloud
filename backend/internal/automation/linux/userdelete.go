package linux

import "fmt"

// DeleteUser removes a Linux user and its home directory.
func DeleteUser(username string) error {

	_, err := Execute(
		"userdel",
		"-r",
		username,
	)

	if err != nil {
		return fmt.Errorf("failed to delete linux user: %w", err)
	}

	return nil
}
