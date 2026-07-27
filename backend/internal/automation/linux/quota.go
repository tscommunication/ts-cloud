package linux

import (
	"fmt"
	"strconv"
)

// SetQuota applies a filesystem quota (GB) to a Linux user.
func SetQuota(username string, quotaGB int) error {

	if quotaGB <= 0 {
		return nil
	}

	// Convert GB ? KB (setquota uses KB)
	quotaKB := quotaGB * 1024 * 1024

	_, err := Execute(
		"sudo",
		"/usr/sbin/setquota",
		"-u",
		username,
		strconv.Itoa(quotaKB),
		strconv.Itoa(quotaKB),
		"0",
		"0",
		"/",
	)

	if err != nil {
		return fmt.Errorf("failed to set quota: %w", err)
	}

	return nil
}

// RemoveQuota removes filesystem quota from a Linux user.
func RemoveQuota(username string) error {

	_, err := Execute(
		"sudo",
		"/usr/sbin/setquota",
		"-u",
		username,
		"0",
		"0",
		"0",
		"0",
		"/",
	)

	if err != nil {
		return fmt.Errorf("failed to remove quota: %w", err)
	}

	return nil
}
