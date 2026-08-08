package linux

import "fmt"

func TestVSFTPDParser() error {

	lines, err := ReadNewVSFTPDLogLines()
	if err != nil {
		return err
	}

	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}
