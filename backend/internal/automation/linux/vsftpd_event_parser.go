package linux

import (
	"regexp"
	"strconv"
)

type VSFTPDEventType string

const (
	EventLoginSuccess VSFTPDEventType = "LOGIN_SUCCESS"
	EventLoginFailed  VSFTPDEventType = "LOGIN_FAILED"
	EventUpload       VSFTPDEventType = "UPLOAD"
	EventDownload     VSFTPDEventType = "DOWNLOAD"
)

type VSFTPDEvent struct {
	Type     VSFTPDEventType
	Username string
	IP       string
	FileName string
	Bytes    int64
}

var (
	userRegex = regexp.MustCompile(
		`\[pid [0-9]+\] \[(.*?)\]`,
	)

	loginRegex = regexp.MustCompile(
		`OK LOGIN: Client "(.*?)"`,
	)

	failRegex = regexp.MustCompile(
		`FAIL LOGIN: Client "(.*?)"`,
	)

	uploadRegex = regexp.MustCompile(
		`OK UPLOAD: Client "(.*?)", "(.*?)", ([0-9]+) bytes`,
	)

	downloadRegex = regexp.MustCompile(
		`OK DOWNLOAD: Client "(.*?)", "(.*?)", ([0-9]+) bytes`,
	)
)

func getUsername(line string) string {

	match := userRegex.FindStringSubmatch(line)

	if len(match) > 1 {
		return match[1]
	}

	return ""
}

func ParseVSFTPDLine(line string) *VSFTPDEvent {

	username := getUsername(line)

	if username == "" {
		return nil
	}

	if uploadRegex.MatchString(line) {

		m := uploadRegex.FindStringSubmatch(line)

		size, _ := strconv.ParseInt(m[3], 10, 64)

		return &VSFTPDEvent{
			Type:     EventUpload,
			Username: username,
			IP:       m[1],
			FileName: m[2],
			Bytes:    size,
		}
	}

	if downloadRegex.MatchString(line) {

		m := downloadRegex.FindStringSubmatch(line)

		size, _ := strconv.ParseInt(m[3], 10, 64)

		return &VSFTPDEvent{
			Type:     EventDownload,
			Username: username,
			IP:       m[1],
			FileName: m[2],
			Bytes:    size,
		}
	}

	if loginRegex.MatchString(line) {

		m := loginRegex.FindStringSubmatch(line)

		return &VSFTPDEvent{
			Type:     EventLoginSuccess,
			Username: username,
			IP:       m[1],
		}
	}

	if failRegex.MatchString(line) {

		m := failRegex.FindStringSubmatch(line)

		return &VSFTPDEvent{
			Type:     EventLoginFailed,
			Username: username,
			IP:       m[1],
		}
	}

	return nil
}
