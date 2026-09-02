package szcom

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	defaultTimeout    = 5 * time.Second
	DefaultTelnetPort = 23
)

var macPattern = regexp.MustCompile(
	`(?i)\b[0-9a-f]{2}(?::[0-9a-f]{2}){5}\b`,
)

type Client struct {
	host     string
	port     int
	username string
	password string
	timeout  time.Duration
}

type LearnedMACResolution struct {
	MACAddress string
	PONNo      int
	ONUNo      int
	ETHPort    int
}

type commandRunner func(
	command string,
) (
	string,
	error,
)

func NewClient(
	host string,
	port int,
	username string,
	password string,
) (*Client, error) {
	host = strings.TrimSpace(host)
	username = strings.TrimSpace(username)

	if host == "" {
		return nil, errors.New("management host is required")
	}

	if port < 1 || port > 65535 {
		return nil, errors.New(
			"management port must be between 1 and 65535",
		)
	}

	if username == "" {
		return nil, errors.New(
			"management username is required",
		)
	}

	if password == "" {
		return nil, errors.New(
			"management password is required",
		)
	}

	return &Client{
		host:     host,
		port:     port,
		username: username,
		password: password,
		timeout:  defaultTimeout,
	}, nil
}

func (c *Client) ResolveLearnedMAC(
	ctx context.Context,
	ponNo int,
	onuNos []int,
	macAddress string,
) (*LearnedMACResolution, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}

	if ponNo <= 0 {
		return nil, errors.New("PON number is required")
	}

	target, err := normalizeMAC(macAddress)
	if err != nil {
		return nil, err
	}

	dialer := net.Dialer{
		Timeout: c.timeout,
	}

	conn, err := dialer.DialContext(
		ctx,
		"tcp",
		net.JoinHostPort(
			c.host,
			fmt.Sprintf("%d", c.port),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"connect SZCOM telnet: %w",
			err,
		)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	if _, err := readUntil(
		ctx,
		conn,
		reader,
		c.timeout,
		"Username:",
	); err != nil {
		return nil, fmt.Errorf(
			"wait SZCOM username prompt: %w",
			err,
		)
	}

	if err := writeLine(conn, c.username); err != nil {
		return nil, fmt.Errorf(
			"send SZCOM username: %w",
			err,
		)
	}

	if _, err := readUntil(
		ctx,
		conn,
		reader,
		c.timeout,
		"Password:",
	); err != nil {
		return nil, fmt.Errorf(
			"wait SZCOM password prompt: %w",
			err,
		)
	}

	if err := writeLine(conn, c.password); err != nil {
		return nil, fmt.Errorf(
			"send SZCOM password: %w",
			err,
		)
	}

	loginOutput, err := readUntilAny(
		ctx,
		conn,
		reader,
		c.timeout,
		">",
		"#",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"wait SZCOM CLI prompt: %w",
			err,
		)
	}

	if strings.HasSuffix(
		strings.TrimSpace(loginOutput),
		">",
	) {
		if err := writeLine(conn, "enable"); err != nil {
			return nil, fmt.Errorf(
				"enter SZCOM privileged mode: %w",
				err,
			)
		}

		if _, err := readUntil(
			ctx,
			conn,
			reader,
			c.timeout,
			"#",
		); err != nil {
			return nil, fmt.Errorf(
				"wait SZCOM privileged prompt: %w",
				err,
			)
		}
	}

	run := func(
		command string,
	) (
		string,
		error,
	) {
		if err := writeLine(conn, command); err != nil {
			return "", err
		}

		return readUntil(
			ctx,
			conn,
			reader,
			c.timeout,
			"#",
		)
	}

	return resolveLearnedMACWithRunner(
		ponNo,
		onuNos,
		target,
		run,
	)
}

func (c *Client) ResolveLearnedMACAcrossPONs(
	ctx context.Context,
	ponONUs map[int][]int,
	macAddress string,
) (*LearnedMACResolution, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}

	target, err := normalizeMAC(macAddress)
	if err != nil {
		return nil, err
	}

	dialer := net.Dialer{
		Timeout: c.timeout,
	}

	conn, err := dialer.DialContext(
		ctx,
		"tcp",
		net.JoinHostPort(
			c.host,
			fmt.Sprintf("%d", c.port),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"connect SZCOM telnet: %w",
			err,
		)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	if _, err := readUntil(
		ctx,
		conn,
		reader,
		c.timeout,
		"Username:",
	); err != nil {
		return nil, fmt.Errorf(
			"wait SZCOM username prompt: %w",
			err,
		)
	}

	if err := writeLine(conn, c.username); err != nil {
		return nil, fmt.Errorf(
			"send SZCOM username: %w",
			err,
		)
	}

	if _, err := readUntil(
		ctx,
		conn,
		reader,
		c.timeout,
		"Password:",
	); err != nil {
		return nil, fmt.Errorf(
			"wait SZCOM password prompt: %w",
			err,
		)
	}

	if err := writeLine(conn, c.password); err != nil {
		return nil, fmt.Errorf(
			"send SZCOM password: %w",
			err,
		)
	}

	loginOutput, err := readUntilAny(
		ctx,
		conn,
		reader,
		c.timeout,
		">",
		"#",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"wait SZCOM CLI prompt: %w",
			err,
		)
	}

	if strings.HasSuffix(
		strings.TrimSpace(loginOutput),
		">",
	) {
		if err := writeLine(conn, "enable"); err != nil {
			return nil, fmt.Errorf(
				"enter SZCOM privileged mode: %w",
				err,
			)
		}

		if _, err := readUntil(
			ctx,
			conn,
			reader,
			c.timeout,
			"#",
		); err != nil {
			return nil, fmt.Errorf(
				"wait SZCOM privileged prompt: %w",
				err,
			)
		}
	}

	run := func(
		command string,
	) (
		string,
		error,
	) {
		if err := writeLine(conn, command); err != nil {
			return "", err
		}

		return readUntil(
			ctx,
			conn,
			reader,
			c.timeout,
			"#",
		)
	}

	return resolveLearnedMACAcrossPONsWithRunner(
		ponONUs,
		target,
		run,
	)
}

func resolveLearnedMACAcrossPONsWithRunner(
	ponONUs map[int][]int,
	target string,
	run commandRunner,
) (*LearnedMACResolution, error) {
	if run == nil {
		return nil, errors.New(
			"SZCOM command runner is required",
		)
	}

	normalized, err := normalizeMAC(target)
	if err != nil {
		return nil, err
	}

	ponNos := make([]int, 0, len(ponONUs))
	for ponNo := range ponONUs {
		if ponNo > 0 {
			ponNos = append(ponNos, ponNo)
		}
	}
	sort.Ints(ponNos)

	var matches []LearnedMACResolution

	for _, ponNo := range ponNos {
		unique := make(map[int]struct{})
		for _, onuNo := range ponONUs[ponNo] {
			if onuNo > 0 && onuNo <= 64 {
				unique[onuNo] = struct{}{}
			}
		}

		onuNos := make([]int, 0, len(unique))
		for onuNo := range unique {
			onuNos = append(onuNos, onuNo)
		}
		sort.Ints(onuNos)

		for _, onuNo := range onuNos {
			for ethPort := 1; ethPort <= 4; ethPort++ {
				command := fmt.Sprintf(
					"show ont port learned-mac pon%d %d eth %d",
					ponNo,
					onuNo,
					ethPort,
				)

				output, err := run(command)
				if err != nil {
					return nil, fmt.Errorf(
						"run SZCOM learned-MAC command for PON %d ONU %d ETH %d: %w",
						ponNo,
						onuNo,
						ethPort,
						err,
					)
				}

				if !outputMatchesTargetMAC(
					output,
					normalized,
				) {
					continue
				}

				matches = append(
					matches,
					LearnedMACResolution{
						MACAddress: normalized,
						PONNo:      ponNo,
						ONUNo:      onuNo,
						ETHPort:    ethPort,
					},
				)
			}
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}

	first := matches[0]

	for _, match := range matches[1:] {
		if match.PONNo != first.PONNo ||
			match.ONUNo != first.ONUNo {
			// Never guess across PONs or ONUs when the firmware's
			// shifted/truncated representation is ambiguous.
			return nil, nil
		}
	}

	return &first, nil
}

func resolveLearnedMACWithRunner(
	ponNo int,
	onuNos []int,
	target string,
	run commandRunner,
) (*LearnedMACResolution, error) {
	if run == nil {
		return nil, errors.New(
			"SZCOM command runner is required",
		)
	}

	normalized, err := normalizeMAC(target)
	if err != nil {
		return nil, err
	}

	unique := make(map[int]struct{})

	for _, onuNo := range onuNos {
		if onuNo > 0 && onuNo <= 64 {
			unique[onuNo] = struct{}{}
		}
	}

	ordered := make([]int, 0, len(unique))
	for onuNo := range unique {
		ordered = append(ordered, onuNo)
	}
	sort.Ints(ordered)

	var matches []LearnedMACResolution

	for _, onuNo := range ordered {
		for ethPort := 1; ethPort <= 4; ethPort++ {
			command := fmt.Sprintf(
				"show ont port learned-mac pon%d %d eth %d",
				ponNo,
				onuNo,
				ethPort,
			)

			output, err := run(command)
			if err != nil {
				return nil, fmt.Errorf(
					"run SZCOM learned-MAC command for PON %d ONU %d ETH %d: %w",
					ponNo,
					onuNo,
					ethPort,
					err,
				)
			}

			if !outputMatchesTargetMAC(
				output,
				normalized,
			) {
				continue
			}

			matches = append(
				matches,
				LearnedMACResolution{
					MACAddress: normalized,
					PONNo:      ponNo,
					ONUNo:      onuNo,
					ETHPort:    ethPort,
				},
			)
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}

	first := matches[0]

	for _, match := range matches[1:] {
		if match.ONUNo != first.ONUNo {
			// Never guess when the firmware's truncated learned-MAC
			// representation produces more than one ONU candidate.
			return nil, nil
		}
	}

	return &first, nil
}

func outputMatchesTargetMAC(
	output string,
	target string,
) bool {
	targetMAC, err := net.ParseMAC(target)
	if err != nil || len(targetMAC) != 6 {
		return false
	}

	for _, raw := range macPattern.FindAllString(
		output,
		-1,
	) {
		found, err := net.ParseMAC(raw)
		if err != nil || len(found) != 6 {
			continue
		}

		if strings.EqualFold(
			found.String(),
			targetMAC.String(),
		) {
			return true
		}

		// Observed FOS CLI quirk:
		// a target such as AA:BB:CC:DD:EE:FF can be rendered as
		//                  00:AA:BB:CC:DD:EE
		//
		// Therefore accept only the exact observed shifted-prefix
		// representation. Ambiguous ONU matches are rejected above.
		if found[0] == 0 &&
			found[1] == targetMAC[0] &&
			found[2] == targetMAC[1] &&
			found[3] == targetMAC[2] &&
			found[4] == targetMAC[3] &&
			found[5] == targetMAC[4] {
			return true
		}
	}

	return false
}

func normalizeMAC(
	value string,
) (
	string,
	error,
) {
	parsed, err := net.ParseMAC(
		strings.TrimSpace(value),
	)
	if err != nil || len(parsed) != 6 {
		return "", errors.New(
			"invalid MAC address",
		)
	}

	return strings.ToUpper(parsed.String()), nil
}

func writeLine(
	conn net.Conn,
	value string,
) error {
	_, err := conn.Write(
		[]byte(value + "\r\n"),
	)
	return err
}

func readUntil(
	ctx context.Context,
	conn net.Conn,
	reader *bufio.Reader,
	timeout time.Duration,
	needle string,
) (
	string,
	error,
) {
	return readUntilAny(
		ctx,
		conn,
		reader,
		timeout,
		needle,
	)
}

func readUntilAny(
	ctx context.Context,
	conn net.Conn,
	reader *bufio.Reader,
	timeout time.Duration,
	needles ...string,
) (
	string,
	error,
) {
	var out strings.Builder

	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		if err := conn.SetReadDeadline(
			time.Now().Add(timeout),
		); err != nil {
			return "", err
		}

		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}

		// Minimal Telnet option negotiation.
		if b == 255 {
			cmd, err := reader.ReadByte()
			if err != nil {
				return "", err
			}

			if cmd == 255 {
				out.WriteByte(255)
				continue
			}

			switch cmd {
			case 251, 252, 253, 254:
				option, err := reader.ReadByte()
				if err != nil {
					return "", err
				}

				reply := byte(254) // DONT
				if cmd == 253 || cmd == 254 {
					reply = 252 // WONT
				}

				_, _ = conn.Write(
					[]byte{255, reply, option},
				)

			case 250:
				// Skip sub-negotiation through IAC SE.
				for {
					x, err := reader.ReadByte()
					if err != nil {
						return "", err
					}
					if x != 255 {
						continue
					}
					y, err := reader.ReadByte()
					if err != nil {
						return "", err
					}
					if y == 240 {
						break
					}
				}
			}

			continue
		}

		out.WriteByte(b)
		current := out.String()

		for _, needle := range needles {
			if strings.Contains(current, needle) {
				return current, nil
			}
		}
	}
}
