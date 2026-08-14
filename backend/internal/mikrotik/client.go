package mikrotik

import (
	"bufio"
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Resource struct {
	Identity    string
	Version     string
	BoardName   string
	Uptime      string
	CPULoad     int
	TotalMemory int64
	FreeMemory  int64
}

type ConnectionError struct{ Err error }

func (err *ConnectionError) Error() string { return "connect to RouterOS API: " + err.Err.Error() }
func (err *ConnectionError) Unwrap() error { return err.Err }

func FetchResource(host string, port int, useTLS bool, username, password string) (Resource, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	var connection net.Conn
	var err error
	if useTLS {
		connection, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host})
	} else {
		connection, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return Resource{}, &ConnectionError{Err: err}
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(8 * time.Second))

	client := &client{reader: bufio.NewReader(connection), writer: bufio.NewWriter(connection)}
	if err := client.login(username, password); err != nil {
		return Resource{}, err
	}
	identityRows, err := client.command("/system/identity/print")
	if err != nil {
		return Resource{}, err
	}
	resourceRows, err := client.command("/system/resource/print")
	if err != nil {
		return Resource{}, err
	}
	var result Resource
	if len(identityRows) > 0 {
		result.Identity = identityRows[0]["name"]
	}
	if len(resourceRows) > 0 {
		row := resourceRows[0]
		result.Version = row["version"]
		result.BoardName = row["board-name"]
		result.Uptime = row["uptime"]
		result.CPULoad, _ = strconv.Atoi(row["cpu-load"])
		result.TotalMemory, _ = strconv.ParseInt(row["total-memory"], 10, 64)
		result.FreeMemory, _ = strconv.ParseInt(row["free-memory"], 10, 64)
	}
	return result, nil
}

type client struct {
	reader *bufio.Reader
	writer *bufio.Writer
}

func (c *client) login(username, password string) error {
	replies, done, err := c.exchange([]string{"/login", "=name=" + username, "=password=" + password})
	if err != nil {
		return fmt.Errorf("RouterOS authentication failed: %w", err)
	}
	challenge := done["ret"]
	if challenge == "" {
		return nil
	}
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return errors.New("RouterOS returned an invalid login challenge")
	}
	hashInput := append([]byte{0}, []byte(password)...)
	hashInput = append(hashInput, challengeBytes...)
	digest := md5.Sum(hashInput) // RouterOS legacy challenge-response protocol.
	_, _, err = c.exchange([]string{"/login", "=name=" + username, "=response=00" + hex.EncodeToString(digest[:])})
	_ = replies
	if err != nil {
		return fmt.Errorf("RouterOS authentication failed: %w", err)
	}
	return nil
}

func (c *client) command(command string) ([]map[string]string, error) {
	rows, _, err := c.exchange([]string{command})
	return rows, err
}

func (c *client) exchange(words []string) ([]map[string]string, map[string]string, error) {
	if err := writeSentence(c.writer, words); err != nil {
		return nil, nil, err
	}
	var rows []map[string]string
	for {
		sentence, err := readSentence(c.reader)
		if err != nil {
			return nil, nil, err
		}
		if len(sentence) == 0 {
			continue
		}
		attributes := sentenceAttributes(sentence[1:])
		switch sentence[0] {
		case "!re":
			rows = append(rows, attributes)
		case "!done":
			return rows, attributes, nil
		case "!trap", "!fatal":
			message := attributes["message"]
			if message == "" {
				message = sentence[0]
			}
			return nil, nil, errors.New(message)
		}
	}
}

func sentenceAttributes(words []string) map[string]string {
	attributes := make(map[string]string)
	for _, word := range words {
		if !strings.HasPrefix(word, "=") {
			continue
		}
		parts := strings.SplitN(word[1:], "=", 2)
		if len(parts) == 2 {
			attributes[parts[0]] = parts[1]
		}
	}
	return attributes
}

func writeSentence(writer *bufio.Writer, words []string) error {
	for _, word := range words {
		if err := writeWord(writer, word); err != nil {
			return err
		}
	}
	if err := writer.WriteByte(0); err != nil {
		return err
	}
	return writer.Flush()
}

func writeWord(writer io.Writer, word string) error {
	if err := writeLength(writer, uint32(len(word))); err != nil {
		return err
	}
	_, err := io.WriteString(writer, word)
	return err
}

func writeLength(writer io.Writer, length uint32) error {
	var encoded [5]byte
	var data []byte
	switch {
	case length < 0x80:
		data = encoded[:1]
		data[0] = byte(length)
	case length < 0x4000:
		data = encoded[:2]
		binary.BigEndian.PutUint16(data, uint16(length)|0x8000)
	case length < 0x200000:
		data = encoded[:3]
		value := length | 0xC00000
		data[0], data[1], data[2] = byte(value>>16), byte(value>>8), byte(value)
	case length < 0x10000000:
		data = encoded[:4]
		binary.BigEndian.PutUint32(data, length|0xE0000000)
	default:
		data = encoded[:5]
		data[0] = 0xF0
		binary.BigEndian.PutUint32(data[1:], length)
	}
	_, err := writer.Write(data)
	return err
}

func readSentence(reader *bufio.Reader) ([]string, error) {
	var sentence []string
	for {
		length, err := readLength(reader)
		if err != nil {
			return nil, err
		}
		if length == 0 {
			return sentence, nil
		}
		word := make([]byte, length)
		if _, err := io.ReadFull(reader, word); err != nil {
			return nil, err
		}
		sentence = append(sentence, string(word))
	}
}

func readLength(reader io.Reader) (uint32, error) {
	var first [1]byte
	if _, err := io.ReadFull(reader, first[:]); err != nil {
		return 0, err
	}
	switch {
	case first[0]&0x80 == 0:
		return uint32(first[0]), nil
	case first[0]&0xC0 == 0x80:
		var rest [1]byte
		_, err := io.ReadFull(reader, rest[:])
		return (uint32(first[0]&0x3F) << 8) | uint32(rest[0]), err
	case first[0]&0xE0 == 0xC0:
		var rest [2]byte
		_, err := io.ReadFull(reader, rest[:])
		return (uint32(first[0]&0x1F) << 16) | uint32(rest[0])<<8 | uint32(rest[1]), err
	case first[0]&0xF0 == 0xE0:
		var rest [3]byte
		_, err := io.ReadFull(reader, rest[:])
		return (uint32(first[0]&0x0F) << 24) | uint32(rest[0])<<16 | uint32(rest[1])<<8 | uint32(rest[2]), err
	case first[0] == 0xF0:
		var rest [4]byte
		_, err := io.ReadFull(reader, rest[:])
		return binary.BigEndian.Uint32(rest[:]), err
	default:
		return 0, errors.New("invalid RouterOS API word length")
	}
}
