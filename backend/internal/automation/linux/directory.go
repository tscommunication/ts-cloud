package linux

import "fmt"

// CreateHomeDirectory creates the user's home directory.
func CreateHomeDirectory(path string) error {

	_, err := Execute(
		"sudo",
		"/usr/bin/mkdir",
		"-p",
		path,
	)

	if err != nil {
		return fmt.Errorf("failed to create home directory: %w", err)
	}

	return nil
}

// ChangeOwner changes ownership of a directory.
func ChangeOwner(path, username string) error {

	_, err := Execute(
		"sudo",
		"/usr/bin/chown",
		"-R",
		fmt.Sprintf("%s:%s", username, username),
		path,
	)

	if err != nil {
		return fmt.Errorf("failed to change owner: %w", err)
	}

	return nil
}

// SetPermissions sets directory permissions.
func SetPermissions(path string) error {

	_, err := Execute(
		"sudo",
		"/usr/bin/chmod",
		"755",
		path,
	)

	if err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}
