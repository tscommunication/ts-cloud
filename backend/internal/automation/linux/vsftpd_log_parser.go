package linux

import (
	"bufio"
	"os"
	"syscall"
)

const VSFTPDLogFile = "/var/log/vsftpd.log"

func ReadNewVSFTPDLogLines(
	lastOffset int64,
	lastInode uint64,
) ([]string, int64, uint64, error) {

	file, err := os.Open(VSFTPDLogFile)
	if err != nil {
		return nil, lastOffset, lastInode, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, lastOffset, lastInode, err
	}

	stat := info.Sys().(*syscall.Stat_t)
	currentInode := uint64(stat.Ino)

	// Log rotated
	if currentInode != lastInode {
		lastOffset = 0
	}

	_, err = file.Seek(lastOffset, 0)
	if err != nil {
		return nil, lastOffset, currentInode, err
	}

	var lines []string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	newOffset, err := file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, lastOffset, currentInode, err
	}

	return lines, newOffset, currentInode, nil
}
