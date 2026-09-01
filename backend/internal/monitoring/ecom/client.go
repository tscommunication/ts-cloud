package ecom

import (
	"bytes"
	"context"
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

const (
	defaultTimeout = 5 * time.Second
	cgiPath        = "/cgi-bin/h.cgi"
)

type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	httpClient *http.Client
}

type LoginResponse struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Data        struct {
		Token string `json:"token"`
	} `json:"data"`
}

type MACAddressRecord struct {
	MacAddress string `json:"MacAddress"`
	PortIndex  string `json:"PortIndex"`
	OnuID      int    `json:"OnuId"`
	VLAN       int    `json:"Vlan"`
	MacType    int    `json:"MacType"`
}

func (r *MACAddressRecord) UnmarshalJSON(data []byte) error {
	if r == nil {
		return errors.New("ECOM MAC record target is nil")
	}

	var raw struct {
		MacAddress string          `json:"MacAddress"`
		PortIndex  string          `json:"PortIndex"`
		OnuID      json.RawMessage `json:"OnuId"`
		VLAN       int             `json:"Vlan"`
		MacType    int             `json:"MacType"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = MACAddressRecord{
		MacAddress: raw.MacAddress,
		PortIndex:  raw.PortIndex,
		VLAN:       raw.VLAN,
		MacType:    raw.MacType,
	}

	if len(raw.OnuID) == 0 || string(raw.OnuID) == "null" {
		return nil
	}

	var numeric int
	if err := json.Unmarshal(raw.OnuID, &numeric); err == nil {
		r.OnuID = numeric
		return nil
	}

	var encoded string
	if err := json.Unmarshal(raw.OnuID, &encoded); err != nil {
		return fmt.Errorf("decode ECOM OnuId: %w", err)
	}

	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil
	}

	parsed, err := strconv.Atoi(encoded)
	if err != nil {
		return fmt.Errorf("decode ECOM OnuId %q: %w", encoded, err)
	}

	r.OnuID = parsed
	return nil
}

type MACAddressRecords []MACAddressRecord

func (records *MACAddressRecords) UnmarshalJSON(data []byte) error {
	if records == nil {
		return errors.New("ECOM MAC records target is nil")
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*records = nil
		return nil
	}

	switch trimmed[0] {
	case '[':
		var rows []MACAddressRecord
		if err := json.Unmarshal(trimmed, &rows); err != nil {
			return err
		}
		*records = rows
		return nil

	case '{':
		var envelope struct {
			MacArray []MACAddressRecord `json:"MacArray"`
		}
		if err := json.Unmarshal(trimmed, &envelope); err != nil {
			return err
		}
		*records = envelope.MacArray
		return nil

	default:
		return errors.New("ECOM MAC data must be an array or object")
	}
}

type MACAddressResponse struct {
	Code        int               `json:"code"`
	Description string            `json:"description"`
	Data        MACAddressRecords `json:"data"`
}

type ExactONUResolution struct {
	MACAddress string
	Interface  string
	PONNo      int
	ONUNo      int
	VLAN       int
	MACType    int
}

type LearnedMACEvidence struct {
	PortID    int
	Interface string
	PONNo     int
	VLAN      int
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
		return nil, errors.New("management port must be between 1 and 65535")
	}
	if username == "" {
		return nil, errors.New("management username is required")
	}
	if password == "" {
		return nil, errors.New("management password is required")
	}

	hostname := host
	if parsed := net.ParseIP(host); parsed != nil && strings.Contains(host, ":") {
		hostname = "[" + host + "]"
	}

	baseURL, err := url.Parse(
		"http://" + hostname + ":" + strconv.Itoa(port),
	)
	if err != nil {
		return nil, fmt.Errorf("build management URL: %w", err)
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

// NewClientWithBaseURL exists for isolated tests and future HTTPS support.
// Production callers should normally use NewClient.
func NewClientWithBaseURL(
	rawBaseURL string,
	username string,
	password string,
	httpClient *http.Client,
) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse management base URL: %w", err)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, errors.New("management URL scheme must be http or https")
	}
	if baseURL.Host == "" {
		return nil, errors.New("management URL host is required")
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("management username is required")
	}
	if password == "" {
		return nil, errors.New("management password is required")
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

func (c *Client) Login(ctx context.Context) (string, error) {
	payload := struct {
		LoginType int    `json:"LoginType"`
		Password  string `json:"Password"`
		Usrname   string `json:"Usrname"`
	}{
		LoginType: 1,
		Password:  c.password,
		Usrname:   c.username,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode ECOM login request: %w", err)
	}

	endpoint := c.endpoint()
	query := endpoint.Query()
	query.Set("module", "sys_login")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create ECOM login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	var response LoginResponse
	if err := c.doJSON(req, &response); err != nil {
		return "", err
	}
	if response.Code != 0 {
		return "", fmt.Errorf(
			"ECOM login failed: code=%d description=%s",
			response.Code,
			strings.TrimSpace(response.Description),
		)
	}

	token := strings.TrimSpace(response.Data.Token)
	if token == "" {
		return "", errors.New("ECOM login response did not include a token")
	}

	return token, nil
}

func (c *Client) FindONUByMAC(
	ctx context.Context,
	token string,
	macAddress string,
) (*ExactONUResolution, error) {
	return c.FindONUByMACWithEvidence(
		ctx,
		token,
		macAddress,
		nil,
	)
}

func (c *Client) FindONUByMACWithEvidence(
	ctx context.Context,
	token string,
	macAddress string,
	evidence *LearnedMACEvidence,
) (*ExactONUResolution, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("ECOM session token is required")
	}

	normalizedMAC, err := normalizeMAC(macAddress)
	if err != nil {
		return nil, err
	}

	endpoint := c.endpoint()
	query := endpoint.Query()
	query.Set("module", "epon_mac_address_get")
	if evidence != nil {
		if evidence.PortID <= 0 {
			return nil, errors.New("ECOM learned MAC PortID is required")
		}
		if evidence.PONNo <= 0 {
			return nil, errors.New("ECOM learned MAC PON is required")
		}
		query.Set("PortIndex", strconv.Itoa(evidence.PortID))
	}
	query.Set("Lag", "1")
	query.Set("CPU", "1")
	query.Set("MacAddr", normalizedMAC)
	query.Set("MacType", "")
	query.Set("OnuId", "")
	query.Set("Vlan", "")
	query.Set("PageNumber", "1")
	query.Set("PageSize", "20")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create ECOM MAC lookup request: %w", err)
	}

	req.Header.Set("token", token)

	var response MACAddressResponse
	if err := c.doJSON(req, &response); err != nil {
		return nil, err
	}
	if response.Code != 0 {
		return nil, fmt.Errorf(
			"ECOM MAC lookup failed: code=%d description=%s",
			response.Code,
			strings.TrimSpace(response.Description),
		)
	}

	for _, record := range response.Data {
		recordMAC, err := normalizeMAC(record.MacAddress)
		if err != nil || recordMAC != normalizedMAC {
			continue
		}

		pon, ok := parsePONInterface(record.PortIndex)
		if !ok {
			continue
		}
		if record.OnuID <= 0 {
			continue
		}

		if evidence != nil {
			evidenceInterface := strings.TrimSpace(
				strings.ToLower(evidence.Interface),
			)
			recordInterface := strings.TrimSpace(
				strings.ToLower(record.PortIndex),
			)

			if evidenceInterface == "" {
				return nil, errors.New(
					"ECOM learned MAC interface is required",
				)
			}
			if recordInterface != evidenceInterface {
				return nil, fmt.Errorf(
					"ECOM HTTP interface %q does not match SNMP interface %q",
					record.PortIndex,
					evidence.Interface,
				)
			}
			if pon != evidence.PONNo {
				return nil, fmt.Errorf(
					"ECOM HTTP PON %d does not match SNMP PON %d",
					pon,
					evidence.PONNo,
				)
			}
			if record.VLAN != evidence.VLAN {
				return nil, fmt.Errorf(
					"ECOM HTTP VLAN %d does not match SNMP VLAN %d",
					record.VLAN,
					evidence.VLAN,
				)
			}
		}

		return &ExactONUResolution{
			MACAddress: recordMAC,
			Interface:  strings.TrimSpace(record.PortIndex),
			PONNo:      pon,
			ONUNo:      record.OnuID,
			VLAN:       record.VLAN,
			MACType:    record.MacType,
		}, nil
	}

	return nil, nil
}

func (c *Client) ResolveONUByMAC(
	ctx context.Context,
	macAddress string,
) (*ExactONUResolution, error) {
	token, err := c.Login(ctx)
	if err != nil {
		return nil, err
	}

	return c.FindONUByMAC(ctx, token, macAddress)
}

func (c *Client) endpoint() *url.URL {
	endpoint := *c.baseURL
	endpoint.Path = cgiPath
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	return &endpoint
}

func (c *Client) doJSON(req *http.Request, target any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ECOM HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("ECOM HTTP status %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode ECOM response: %w", err)
	}

	return nil
}

func normalizeMAC(value string) (string, error) {
	parsed, err := net.ParseMAC(strings.TrimSpace(value))
	if err != nil || len(parsed) != 6 {
		return "", errors.New("invalid MAC address")
	}

	return strings.ToUpper(parsed.String()), nil
}

func parsePONInterface(value string) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))

	var shelf, slot, pon int
	if _, err := fmt.Sscanf(
		value,
		"epon %d/%d/%d",
		&shelf,
		&slot,
		&pon,
	); err != nil {
		return 0, false
	}

	// Reject extra tokens such as "epon 0/1/1 onu 1".
	expected := fmt.Sprintf("epon %d/%d/%d", shelf, slot, pon)
	if value != expected || pon <= 0 {
		return 0, false
	}

	return pon, true
}
