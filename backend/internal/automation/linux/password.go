package linux

import "fmt"

// SetPassword sets a Linux user's password using chpasswd.
func SetPassword(username, password string) error {

	input := fmt.Sprintf("%s:%s\n", username, password)

	_, err := ExecuteWithInput(
		input,
		"sudo",
		"/usr/sbin/chpasswd",
	)

	if err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	return nil
}
