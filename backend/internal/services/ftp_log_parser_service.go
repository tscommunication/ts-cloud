package services

import (
	"bufio"
	"os"
)

const vsftpdLogFile = "/var/log/vsftpd.log"

type FTPLogParser struct {
	logFile string
}

func NewFTPLogParser() *FTPLogParser {

	return &FTPLogParser{
		logFile: vsftpdLogFile,
	}
}

// Parse reads vsftpd log file.
//
// Sprint 14G-3A:
// - Success Login
// - Failed Login
// - Upload
// - Download
func (p *FTPLogParser) Parse() error {

	file, err := os.Open(p.logFile)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		line := scanner.Text()

		// TODO:
		// Parse Login
		// Parse Upload
		// Parse Download

		_ = line
	}

	return scanner.Err()
}
