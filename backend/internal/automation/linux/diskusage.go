package linux

import (
	"fmt"
	"strconv"
	"strings"
)

// GetDirectorySize returns the total size of a directory in bytes.
func GetDirectorySize(path string) (uint64, error) {

	result, err := Execute(
		"du",
		"-sb",
		path,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory size: %w", err)
	}

	output := strings.TrimSpace(result.Stdout)

	if output == "" {
		return 0, fmt.Errorf("empty du output")
	}

	fields := strings.Fields(output)

	if len(fields) == 0 {
		return 0, fmt.Errorf("invalid du output")
	}

	size, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid directory size: %w", err)
	}

	return size, nil
}
