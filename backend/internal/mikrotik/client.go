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
	Identity      string
	Version       string
	BoardName     string
	Uptime        string
	CPULoad       int
	TotalMemory   int64
	FreeMemory    int64
	PPPoESessions []PPPoESession
	PPPSecrets    []PPPSecret
}

type PPPoESession struct {
	Name      string
	Interface string
	Service   string
	CallerID  string
	Address   string
	Uptime    string
	SessionID string
	RxRateBps int64
	TxRateBps int64
	RxBytes   int64
	TxBytes   int64
}

type PPPInterfaceTraffic struct {
	DownloadBps int64
	UploadBps   int64
}

type PPPSecret struct {
	ID            string
	Name          string
	Service       string
	Profile       string
	CallerID      string
	RemoteAddress string
	Disabled      bool
}

type PPPSecretInput struct {
	Name          string
	Password      string
	Service       string
	Profile       string
	CallerID      string
	RemoteAddress string
	Disabled      bool
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
	// Some supported RouterOS releases reject the CLI-only `stats` print flag
	// when it is sent through the API. Request the standard active-session
	// properties instead, so a failed optional statistic can never stop PPP
	// session synchronisation.
	pppoeRows, _, err := client.commandWords(
		"/ppp/active/print",
		"=.proplist=.id,name,interface,service,caller-id,address,uptime,session-id,rx-rate,tx-rate,bytes",
	)
	if err != nil {
		return Resource{}, fmt.Errorf("read active PPP sessions: %w", err)
	}
	// PPP active has no portable live-rate field. Monitor all interfaces once
	// and match the dynamic `pppoe-<username>` interface for each session.
	// This command is optional: a router that does not support it must still
	// complete its normal session sync.
	trafficRows, _, trafficErr := client.commandWords(
		"/interface/monitor-traffic",
		"=interface=all",
		"=once=",
	)
	trafficByInterface := make(map[string][2]int64)
	if trafficErr == nil {
		for _, row := range trafficRows {
			trafficByInterface[normalizeInterfaceName(row["name"])] = [2]int64{
				parseRouterOSBits(row["tx-bits-per-second"]),
				parseRouterOSBits(row["rx-bits-per-second"]),
			}
		}
	}
	secretRows, err := client.command("/ppp/secret/print")
	if err != nil {
		return Resource{}, fmt.Errorf("read PPP secrets: %w", err)
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
	result.PPPoESessions = make([]PPPoESession, 0, len(pppoeRows))
	for _, row := range pppoeRows {
		sessionID := row["session-id"]
		if sessionID == "" {
			sessionID = row[".id"]
		}
		session := PPPoESession{
			Name: row["name"], Interface: row["interface"], Service: row["service"], CallerID: row["caller-id"],
			Address: row["address"], Uptime: row["uptime"], SessionID: sessionID,
			RxRateBps: parseRouterOSRate(row["rx-rate"]), TxRateBps: parseRouterOSRate(row["tx-rate"]),
			RxBytes: parseRouterOSCounterPair(row["bytes"])[0], TxBytes: parseRouterOSCounterPair(row["bytes"])[1],
		}
		if rates, ok := trafficByInterface[normalizeInterfaceName(session.Interface)]; ok && session.Interface != "" {
			session.RxRateBps, session.TxRateBps = rates[0], rates[1]
		} else if rates, ok := trafficByInterface[normalizeInterfaceName("<pppoe-"+session.Name+">")]; ok {
			session.RxRateBps, session.TxRateBps = rates[0], rates[1]
		} else if rates, ok := trafficByInterface[normalizeInterfaceName("pppoe-"+session.Name)]; ok {
			session.RxRateBps, session.TxRateBps = rates[0], rates[1]
		} else if rates, ok := trafficByInterface[normalizeInterfaceName(session.Name)]; ok {
			session.RxRateBps, session.TxRateBps = rates[0], rates[1]
		}
		result.PPPoESessions = append(result.PPPoESessions, session)
	}
	result.PPPSecrets = make([]PPPSecret, 0, len(secretRows))
	for _, row := range secretRows {
		result.PPPSecrets = append(result.PPPSecrets, PPPSecret{ID: row[".id"], Name: row["name"], Service: row["service"], Profile: row["profile"], CallerID: row["caller-id"], RemoteAddress: row["remote-address"], Disabled: strings.EqualFold(row["disabled"], "true")})
	}
	return result, nil
}

// FetchPPPInterfaceTraffic reads a single active PPPoE interface. It is used
// only while an operator has the Live Traffic panel open.
func FetchPPPInterfaceTraffic(host string, port int, useTLS bool, username, password, pppoeUsername string) (PPPInterfaceTraffic, error) {
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
		return PPPInterfaceTraffic{}, &ConnectionError{Err: err}
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(8 * time.Second))
	client := &client{reader: bufio.NewReader(connection), writer: bufio.NewWriter(connection)}
	if err := client.login(username, password); err != nil {
		return PPPInterfaceTraffic{}, err
	}
	interfaceName := "<pppoe-" + strings.TrimSpace(pppoeUsername) + ">"
	rows, _, err := client.commandWords("/interface/monitor-traffic", "=interface="+interfaceName, "=once=")
	if err != nil {
		return PPPInterfaceTraffic{}, fmt.Errorf("monitor PPPoE traffic: %w", err)
	}
	if len(rows) == 0 {
		return PPPInterfaceTraffic{}, fmt.Errorf("PPPoE interface %q is not active", interfaceName)
	}
	row := rows[0]
	return PPPInterfaceTraffic{DownloadBps: parseRouterOSBits(row["tx-bits-per-second"]), UploadBps: parseRouterOSBits(row["rx-bits-per-second"])}, nil
}

// ResolveMACByIP performs a read-only RouterOS ARP lookup for one IP address.
// It never modifies the router. Only complete, valid IP/MAC entries are returned.
func ResolveMACByIP(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	ipAddress string,
) (string, error) {
	ipAddress = strings.TrimSpace(ipAddress)
	if net.ParseIP(ipAddress) == nil {
		return "", fmt.Errorf("IP address is invalid")
	}

	var macAddress string
	err := withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			rows, _, err := c.commandWords(
				"/ip/arp/print",
				"=.proplist=address,mac-address,complete,invalid,disabled",
				"?address="+ipAddress,
			)
			if err != nil {
				return fmt.Errorf("read RouterOS ARP entry: %w", err)
			}

			macAddress = resolveMACFromARPRows(rows, ipAddress)
			return nil
		},
	)
	if err != nil {
		return "", err
	}

	return macAddress, nil
}

func resolveMACFromARPRows(rows []map[string]string, ipAddress string) string {
	ipAddress = strings.TrimSpace(ipAddress)

	for _, row := range rows {
		if strings.TrimSpace(row["address"]) != ipAddress {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row["invalid"]), "true") ||
			strings.EqualFold(strings.TrimSpace(row["disabled"]), "true") ||
			strings.EqualFold(strings.TrimSpace(row["complete"]), "false") {
			continue
		}

		rawMAC := strings.TrimSpace(row["mac-address"])
		if rawMAC == "" {
			continue
		}

		parsedMAC, err := net.ParseMAC(rawMAC)
		if err != nil {
			continue
		}

		return strings.ToUpper(parsedMAC.String())
	}

	return ""
}

func normalizeInterfaceName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseRouterOSBits(value string) int64 {
	value = strings.TrimSpace(value)
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed >= 0 {
		return parsed
	}
	return parseRouterOSRate(value)
}

func parseRouterOSRate(value string) int64 {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, unit := range []struct {
		suffix     string
		multiplier float64
	}{{"gbps", 1_000_000_000}, {"mbps", 1_000_000}, {"kbps", 1_000}, {"bps", 1}} {
		if strings.HasSuffix(value, unit.suffix) {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, unit.suffix)), 64)
			if err != nil || parsed < 0 {
				return 0
			}
			return int64(parsed * unit.multiplier)
		}
	}
	return 0
}

func parseRouterOSCounterPair(value string) [2]int64 {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 {
		return [2]int64{}
	}
	var result [2]int64
	for i := range result {
		parsed, err := strconv.ParseInt(strings.TrimSpace(parts[i]), 10, 64)
		if err == nil && parsed >= 0 {
			result[i] = parsed
		}
	}
	return result
}

type client struct {
	reader *bufio.Reader
	writer *bufio.Writer
}

func withAuthenticatedClient(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	operation func(*client) error,
) error {
	address := net.JoinHostPort(
		host,
		strconv.Itoa(port),
	)

	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	var (
		connection net.Conn
		err        error
	)

	if useTLS {
		connection, err = tls.DialWithDialer(
			dialer,
			"tcp",
			address,
			&tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: host,
			},
		)
	} else {
		connection, err = dialer.Dial(
			"tcp",
			address,
		)
	}

	if err != nil {
		return &ConnectionError{
			Err: err,
		}
	}
	defer connection.Close()

	_ = connection.SetDeadline(
		time.Now().Add(8 * time.Second),
	)

	c := &client{
		reader: bufio.NewReader(connection),
		writer: bufio.NewWriter(connection),
	}

	if err := c.login(
		username,
		password,
	); err != nil {
		return err
	}

	if operation == nil {
		return errors.New(
			"RouterOS operation is required",
		)
	}

	return operation(c)
}

func ListPPPSecrets(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	name string,
) ([]PPPSecret, error) {
	var result []PPPSecret

	err := withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			rows, err := c.listPPPSecrets(name)
			if err != nil {
				return err
			}

			result = rows
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func AddPPPSecret(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	input PPPSecretInput,
) (string, error) {
	var id string

	err := withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			createdID, err := c.addPPPSecret(input)
			if err != nil {
				return err
			}

			id = createdID
			return nil
		},
	)
	if err != nil {
		return "", err
	}

	return id, nil
}

func SetPPPSecret(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	id string,
	input PPPSecretInput,
) error {
	return withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			return c.setPPPSecret(
				id,
				input,
			)
		},
	)
}

func EnablePPPSecret(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	id string,
) error {
	return withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			return c.enablePPPSecret(id)
		},
	)
}

func DisablePPPSecret(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	id string,
) error {
	return withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			return c.disablePPPSecret(id)
		},
	)
}

// DisconnectPPPActiveSessions removes every active PPP session for username.
// Disabling a PPP secret only blocks a new login; RouterOS keeps an already
// authenticated session alive until it is explicitly removed.
func DisconnectPPPActiveSessions(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	pppoeUsername string,
) error {
	return withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			return c.disconnectPPPActiveSessions(pppoeUsername)
		},
	)
}

func RemovePPPSecret(
	host string,
	port int,
	useTLS bool,
	username string,
	password string,
	id string,
) error {
	return withAuthenticatedClient(
		host,
		port,
		useTLS,
		username,
		password,
		func(c *client) error {
			return c.removePPPSecret(id)
		},
	)
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

func (c *client) commandWords(
	words ...string,
) ([]map[string]string, map[string]string, error) {
	if len(words) == 0 {
		return nil, nil, errors.New(
			"RouterOS command is required",
		)
	}

	return c.exchange(words)
}

func (c *client) listPPPSecrets(
	name string,
) ([]PPPSecret, error) {
	words := []string{
		"/ppp/secret/print",
		"=.proplist=.id,name,service,profile,caller-id,remote-address,disabled",
	}

	name = strings.TrimSpace(name)

	rows, _, err := c.commandWords(words...)
	if err != nil {
		return nil, fmt.Errorf(
			"list PPP secrets: %w",
			err,
		)
	}

	result := make(
		[]PPPSecret,
		0,
		len(rows),
	)

	for _, row := range rows {
		if name != "" && !strings.EqualFold(strings.TrimSpace(row["name"]), name) {
			continue
		}

		result = append(
			result,
			PPPSecret{
				ID:            row[".id"],
				Name:          row["name"],
				Service:       row["service"],
				Profile:       row["profile"],
				CallerID:      row["caller-id"],
				RemoteAddress: row["remote-address"],
				Disabled: strings.EqualFold(
					row["disabled"],
					"true",
				) || strings.EqualFold(
					row["disabled"],
					"yes",
				),
			},
		)
	}

	return result, nil
}

func (c *client) addPPPSecret(
	input PPPSecretInput,
) (string, error) {
	if err := validatePPPSecretInput(input); err != nil {
		return "", err
	}

	disabled := "no"
	if input.Disabled {
		disabled = "yes"
	}

	words := []string{
		"/ppp/secret/add",
		"=name=" + strings.TrimSpace(input.Name),
		"=password=" + input.Password,
		"=service=" + normalizedPPPService(
			input.Service,
		),
		"=profile=" + strings.TrimSpace(input.Profile),
	}
	if callerID := strings.TrimSpace(input.CallerID); callerID != "" {
		words = append(words, "=caller-id="+callerID)
	}
	if remoteAddress := strings.TrimSpace(input.RemoteAddress); remoteAddress != "" {
		words = append(words, "=remote-address="+remoteAddress)
	}
	words = append(words, "=disabled="+disabled)

	_, done, err := c.commandWords(words...)

	if err != nil {
		return "", fmt.Errorf(
			"add PPP secret: %w",
			err,
		)
	}

	return done["ret"], nil
}

func (c *client) setPPPSecret(
	id string,
	input PPPSecretInput,
) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return errors.New(
			"PPP secret id is required",
		)
	}

	if err := validatePPPSecretInput(input); err != nil {
		return err
	}

	disabled := "no"
	if input.Disabled {
		disabled = "yes"
	}

	words := []string{
		"/ppp/secret/set",
		"=.id=" + id,
		"=name=" + strings.TrimSpace(input.Name),
		"=password=" + input.Password,
		"=service=" + normalizedPPPService(
			input.Service,
		),
		"=profile=" + strings.TrimSpace(input.Profile),
	}
	if callerID := strings.TrimSpace(input.CallerID); callerID != "" {
		words = append(words, "=caller-id="+callerID)
	}
	if remoteAddress := strings.TrimSpace(input.RemoteAddress); remoteAddress != "" {
		words = append(words, "=remote-address="+remoteAddress)
	}
	words = append(words, "=disabled="+disabled)

	_, _, err := c.commandWords(words...)

	if err != nil {
		return fmt.Errorf(
			"set PPP secret: %w",
			err,
		)
	}

	return nil
}

func (c *client) enablePPPSecret(
	id string,
) error {
	return c.setPPPSecretDisabled(
		id,
		false,
	)
}

func (c *client) disablePPPSecret(
	id string,
) error {
	return c.setPPPSecretDisabled(
		id,
		true,
	)
}

func (c *client) setPPPSecretDisabled(
	id string,
	disabled bool,
) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return errors.New(
			"PPP secret id is required",
		)
	}

	value := "no"
	if disabled {
		value = "yes"
	}

	_, _, err := c.commandWords(
		"/ppp/secret/set",
		"=.id="+id,
		"=disabled="+value,
	)

	if err != nil {
		return fmt.Errorf(
			"update PPP secret state: %w",
			err,
		)
	}

	return nil
}

func (c *client) removePPPSecret(
	id string,
) error {
	id = strings.TrimSpace(id)

	if id == "" {
		return errors.New(
			"PPP secret id is required",
		)
	}

	_, _, err := c.commandWords(
		"/ppp/secret/remove",
		"=.id="+id,
	)

	if err != nil {
		return fmt.Errorf(
			"remove PPP secret: %w",
			err,
		)
	}

	return nil
}

func (c *client) disconnectPPPActiveSessions(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("PPP username is required")
	}

	rows, _, err := c.commandWords(
		"/ppp/active/print",
		"=.proplist=.id,name",
	)
	if err != nil {
		return fmt.Errorf("list active PPP sessions: %w", err)
	}

	for _, row := range rows {
		if !strings.EqualFold(strings.TrimSpace(row["name"]), username) {
			continue
		}

		id := strings.TrimSpace(row[".id"])
		if id == "" {
			return errors.New("active PPP session is missing internal id")
		}
		if _, _, err := c.commandWords("/ppp/active/remove", "=.id="+id); err != nil {
			return fmt.Errorf("remove active PPP session: %w", err)
		}
	}

	return nil
}

func validatePPPSecretInput(
	input PPPSecretInput,
) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New(
			"PPP secret name is required",
		)
	}

	if input.Password == "" {
		return errors.New(
			"PPP secret password is required",
		)
	}

	if strings.TrimSpace(input.Profile) == "" {
		return errors.New(
			"PPP secret profile is required",
		)
	}

	service := normalizedPPPService(
		input.Service,
	)

	if service != "pppoe" {
		return fmt.Errorf(
			"unsupported PPP secret service %q",
			service,
		)
	}

	return nil
}

func normalizedPPPService(
	service string,
) string {
	service = strings.ToLower(
		strings.TrimSpace(service),
	)

	if service == "" {
		return "pppoe"
	}

	return service
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
		case "!done", "!empty":
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
