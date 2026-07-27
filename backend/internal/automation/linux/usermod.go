package linux

import "fmt"

// LockUser locks a Linux user account.
func LockUser(username string) error {

	_, err := Execute(
		"sudo",
		"/usr/sbin/usermod",
		"-L",
		username,
	)

	if err != nil {
		return fmt.Errorf("failed to lock linux user: %w", err)
	}

	return nil
}

// UnlockUser unlocks a Linux user account.
func UnlockUser(username string) error {

	_, err := Execute(
		"sudo",
		"/usr/sbin/usermod",
		"-U",
		username,
	)

	if err != nil {
		return fmt.Errorf("failed to unlock linux user: %w", err)
	}

	return nil
}
