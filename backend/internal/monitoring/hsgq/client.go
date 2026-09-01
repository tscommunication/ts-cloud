package hsgq

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	httpClient *http.Client
}

type responseEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type PONMACRecord struct {
	MACAddress string `json:"macaddr"`
	VLANID     int    `json:"vlan_id"`
	PortID     int    `json:"port_id"`
	ONUID      int    `json:"onu_id"`
	MACType    int    `json:"mac_type"`
	ONUName    string `json:"onu_name"`
}

type LearnedMACResolution struct {
	MACAddress string
	VLANID     int
	PONNo      int
	ONUNo      int
	MACType    int
	ONUName    string
}

func NewClient(
	host string,
	port int,
	username string,
	password string,
	httpClient *http.Client,
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
		return nil, errors.New("management username is required")
	}
	if password == "" {
		return nil, errors.New("management password is required")
	}

	hostname := host
	if parsed := net.ParseIP(host); parsed != nil &&
		strings.Contains(host, ":") {
		hostname = "[" + host + "]"
	}

	baseURL, err := url.Parse(
		"http://" + hostname + ":" + strconv.Itoa(port),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"build HSGQ management URL: %w",
			err,
		)
	}

	return newClient(
		baseURL,
		username,
		password,
		httpClient,
	)
}

func NewClientWithBaseURL(
	rawBaseURL string,
	username string,
	password string,
	httpClient *http.Client,
) (*Client, error) {
	baseURL, err := url.Parse(
		strings.TrimSpace(rawBaseURL),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse HSGQ management URL: %w",
			err,
		)
	}

	if baseURL.Scheme != "http" &&
		baseURL.Scheme != "https" {
		return nil, errors.New(
			"management URL scheme must be http or https",
		)
	}
	if baseURL.Host == "" {
		return nil, errors.New(
			"management URL host is required",
		)
	}

	return newClient(
		baseURL,
		username,
		password,
		httpClient,
	)
}

func newClient(
	baseURL *url.URL,
	username string,
	password string,
	httpClient *http.Client,
) (*Client, error) {
	username = strings.TrimSpace(username)

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

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultTimeout,
		}
	}

	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: httpClient,
	}, nil
}

func (c *Client) Login(
	ctx context.Context,
) (string, error) {
	if ctx == nil {
		return "", errors.New("context is required")
	}

	hash := md5.Sum(
		[]byte(c.username + ":" + c.password),
	)

	payload := map[string]any{
		"method": "set",
		"param": map[string]string{
			"name":      c.username,
			"key":       hex.EncodeToString(hash[:]),
			"value":     base64.StdEncoding.EncodeToString([]byte(c.password)),
			"captcha_v": "",
			"captcha_f": "",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(
			"encode HSGQ login request: %w",
			err,
		)
	}

	endpoint := c.resolve("/userlogin")
	query := endpoint.Query()
	query.Set("form", "login")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf(
			"create HSGQ login request: %w",
			err,
		)
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(
			"HSGQ login request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {
		return "", fmt.Errorf(
			"HSGQ login HTTP status %d",
			resp.StatusCode,
		)
	}

	var response responseEnvelope

	decoder := json.NewDecoder(
		io.LimitReader(resp.Body, 4<<20),
	)
	if err := decoder.Decode(&response); err != nil {
		return "", fmt.Errorf(
			"decode HSGQ login response: %w",
			err,
		)
	}

	if response.Code != 1 &&
		response.Code != 2 {
		return "", fmt.Errorf(
			"HSGQ login failed: code=%d message=%s",
			response.Code,
			strings.TrimSpace(response.Message),
		)
	}

	token := strings.TrimSpace(
		resp.Header.Get("X-Token"),
	)
	if token == "" {
		return "", errors.New(
			"HSGQ login response did not include X-Token",
		)
	}

	return token, nil
}

func (c *Client) PreparePONMACTable(
	ctx context.Context,
	token string,
) error {
	var response responseEnvelope

	if err := c.getJSON(
		ctx,
		token,
		"/pon_mac?form=table",
		&response,
	); err != nil {
		return err
	}

	if response.Code != 1 {
		return fmt.Errorf(
			"HSGQ PON MAC prepare failed: code=%d message=%s",
			response.Code,
			strings.TrimSpace(response.Message),
		)
	}

	return nil
}

func (c *Client) FetchPONMACTable(
	ctx context.Context,
	token string,
) ([]PONMACRecord, error) {
	endpoint := "/pon_mac_table?t=" +
		strconv.FormatInt(
			time.Now().UnixMilli(),
			10,
		)

	var response struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    []PONMACRecord `json:"data"`
	}

	if err := c.getJSON(
		ctx,
		token,
		endpoint,
		&response,
	); err != nil {
		return nil, err
	}

	if response.Code != 1 {
		return nil, fmt.Errorf(
			"HSGQ PON MAC fetch failed: code=%d message=%s",
			response.Code,
			strings.TrimSpace(response.Message),
		)
	}

	return response.Data, nil
}

func (c *Client) ResolveLearnedMAC(
	ctx context.Context,
	token string,
	macAddress string,
) (*LearnedMACResolution, error) {
	target, err := normalizeMAC(macAddress)
	if err != nil {
		return nil, err
	}

	if err := c.PreparePONMACTable(
		ctx,
		token,
	); err != nil {
		return nil, err
	}

	rows, err := c.FetchPONMACTable(
		ctx,
		token,
	)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		recordMAC, err := normalizeMAC(
			row.MACAddress,
		)
		if err != nil ||
			recordMAC != target {
			continue
		}

		if row.PortID <= 0 ||
			row.ONUID <= 0 {
			return nil, fmt.Errorf(
				"HSGQ learned MAC has invalid PON/ONU: pon=%d onu=%d",
				row.PortID,
				row.ONUID,
			)
		}

		return &LearnedMACResolution{
			MACAddress: recordMAC,
			VLANID:     row.VLANID,
			PONNo:      row.PortID,
			ONUNo:      row.ONUID,
			MACType:    row.MACType,
			ONUName:    strings.TrimSpace(row.ONUName),
		}, nil
	}

	return nil, nil
}

func (c *Client) getJSON(
	ctx context.Context,
	token string,
	path string,
	target any,
) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New(
			"HSGQ session token is required",
		)
	}

	endpoint := c.resolve(path)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return fmt.Errorf(
			"create HSGQ request: %w",
			err,
		)
	}

	req.Header.Set("X-Token", token)

	return c.doJSON(req, target)
}

func (c *Client) doJSON(
	req *http.Request,
	target any,
) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf(
			"HSGQ management request: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {
		return fmt.Errorf(
			"HSGQ management HTTP status %d",
			resp.StatusCode,
		)
	}

	decoder := json.NewDecoder(
		io.LimitReader(resp.Body, 4<<20),
	)

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf(
			"decode HSGQ management response: %w",
			err,
		)
	}

	return nil
}

func (c *Client) resolve(
	path string,
) *url.URL {
	endpoint := *c.baseURL

	parsed, err := url.Parse(path)
	if err == nil {
		endpoint.Path = parsed.Path
		endpoint.RawQuery = parsed.RawQuery
	}

	return &endpoint
}

func normalizeMAC(
	value string,
) (string, error) {
	var b strings.Builder

	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'a' && r <= 'f':
			b.WriteRune(r)
		case r >= 'A' && r <= 'F':
			b.WriteRune(r + ('a' - 'A'))
		}
	}

	compact := b.String()
	if len(compact) != 12 {
		return "", fmt.Errorf(
			"invalid MAC address %q",
			value,
		)
	}

	return compact[0:2] + ":" +
		compact[2:4] + ":" +
		compact[4:6] + ":" +
		compact[6:8] + ":" +
		compact[8:10] + ":" +
		compact[10:12], nil
}
