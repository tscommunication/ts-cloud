package ecom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveONUByMAC(t *testing.T) {
	const (
		username = "olt-admin"
		password = "management-password"
		token    = "session-token"
		target   = "04:95:E6:58:8E:E8"
	)

	var loginSeen bool
	var lookupSeen bool

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		switch r.URL.Query().Get("module") {
		case "sys_login":
			loginSeen = true

			if r.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("token"); got != "" {
				t.Fatalf("pre-login token header = %q, want blank", got)
			}

			var payload struct {
				LoginType int    `json:"LoginType"`
				Password  string `json:"Password"`
				Usrname   string `json:"Usrname"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.LoginType != 1 {
				t.Fatalf("LoginType = %d, want 1", payload.LoginType)
			}
			if payload.Usrname != username {
				t.Fatalf("Usrname = %q, want expected username", payload.Usrname)
			}
			if payload.Password != password {
				t.Fatal("login password did not match expected value")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"code":0,"description":"success.","data":{"token":"session-token"}}`,
			))

		case "epon_mac_address_get":
			lookupSeen = true

			if r.Method != http.MethodGet {
				t.Fatalf("lookup method = %s, want GET", r.Method)
			}
			if got := r.Header.Get("token"); got != token {
				t.Fatalf("token header = %q, want expected token", got)
			}
			if got := r.URL.Query().Get("MacAddr"); got != target {
				t.Fatalf("MacAddr = %q, want %q", got, target)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"code":0,"description":"success.","data":[` +
					`{"MacAddress":"04:95:E6:58:8E:E8",` +
					`"PortIndex":"epon 0/1/1","OnuId":1,` +
					`"Vlan":3501,"MacType":1}` +
					`]}`,
			))

		default:
			http.Error(w, "unexpected module", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(
		server.URL,
		username,
		password,
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	resolution, err := client.ResolveONUByMAC(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if resolution == nil {
		t.Fatal("expected exact ONU resolution")
	}

	if !loginSeen {
		t.Fatal("expected login request")
	}
	if !lookupSeen {
		t.Fatal("expected MAC lookup request")
	}

	if resolution.MACAddress != target {
		t.Fatalf(
			"MACAddress = %q, want %q",
			resolution.MACAddress,
			target,
		)
	}
	if resolution.Interface != "epon 0/1/1" {
		t.Fatalf(
			"Interface = %q, want epon 0/1/1",
			resolution.Interface,
		)
	}
	if resolution.PONNo != 1 {
		t.Fatalf("PONNo = %d, want 1", resolution.PONNo)
	}
	if resolution.ONUNo != 1 {
		t.Fatalf("ONUNo = %d, want 1", resolution.ONUNo)
	}
	if resolution.VLAN != 3501 {
		t.Fatalf("VLAN = %d, want 3501", resolution.VLAN)
	}
	if resolution.MACType != 1 {
		t.Fatalf("MACType = %d, want 1", resolution.MACType)
	}
}

func TestFindONUByMACNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Query().Get("module") != "epon_mac_address_get" {
			http.Error(w, "unexpected module", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"code":0,"description":"success.","data":[]}`,
		))
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(
		server.URL,
		"user",
		"password",
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	resolution, err := client.FindONUByMAC(
		context.Background(),
		"session-token",
		"04:95:e6:58:8e:e8",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolution != nil {
		t.Fatalf("resolution = %#v, want nil", resolution)
	}
}

func TestLoginRejectsMissingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"code":0,"description":"success.","data":{}}`,
		))
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(
		server.URL,
		"user",
		"password",
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Login(context.Background())
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("Login error = %v, want missing-token error", err)
	}
}

func TestFindONUByMACRejectsNonPONRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"code":0,"description":"success.","data":[` +
				`{"MacAddress":"04:95:E6:58:8E:E8",` +
				`"PortIndex":"epon 0/1/1 onu 1","OnuId":1,` +
				`"Vlan":3501,"MacType":1}` +
				`]}`,
		))
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(
		server.URL,
		"user",
		"password",
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	resolution, err := client.FindONUByMAC(
		context.Background(),
		"session-token",
		"04:95:E6:58:8E:E8",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolution != nil {
		t.Fatalf("resolution = %#v, want nil", resolution)
	}
}

func TestFindONUByMACWithEvidenceUsesProvenQueryAndValidatesResponse(
	t *testing.T,
) {
	const target = "04:95:E6:58:8E:E8"

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		query := r.URL.Query()

		if got := query.Get("module"); got != "epon_mac_address_get" {
			t.Fatalf("module = %q", got)
		}
		if got := query.Get("PortIndex"); got != "17825793" {
			t.Fatalf("PortIndex = %q, want 17825793", got)
		}
		if got := query.Get("Lag"); got != "1" {
			t.Fatalf("Lag = %q, want 1", got)
		}
		if got := query.Get("CPU"); got != "1" {
			t.Fatalf("CPU = %q, want 1", got)
		}
		if got := query.Get("MacAddr"); got != target {
			t.Fatalf("MacAddr = %q, want %q", got, target)
		}
		if got := query.Get("MacType"); got != "" {
			t.Fatalf("MacType = %q, want blank", got)
		}
		if got := query.Get("OnuId"); got != "" {
			t.Fatalf("OnuId = %q, want blank", got)
		}
		if got := query.Get("Vlan"); got != "" {
			t.Fatalf("Vlan = %q, want blank", got)
		}
		if got := query.Get("PageNumber"); got != "1" {
			t.Fatalf("PageNumber = %q, want 1", got)
		}
		if got := query.Get("PageSize"); got != "20" {
			t.Fatalf("PageSize = %q, want 20", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"code":0,"description":"success.","data":[` +
				`{"MacAddress":"04:95:E6:58:8E:E8",` +
				`"PortIndex":"epon 0/1/1","OnuId":1,` +
				`"Vlan":3501,"MacType":1}` +
				`]}`,
		))
	}))
	defer server.Close()

	client, err := NewClientWithBaseURL(
		server.URL,
		"user",
		"password",
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}

	resolution, err := client.FindONUByMACWithEvidence(
		context.Background(),
		"session-token",
		target,
		&LearnedMACEvidence{
			PortID:    17825793,
			Interface: "epon 0/1/1",
			PONNo:     1,
			VLAN:      3501,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolution == nil {
		t.Fatal("expected exact ONU resolution")
	}
	if resolution.ONUNo != 1 {
		t.Fatalf("ONUNo = %d, want 1", resolution.ONUNo)
	}
}

func TestFindONUByMACWithEvidenceRejectsSNMPMismatch(t *testing.T) {
	tests := []struct {
		name     string
		evidence LearnedMACEvidence
	}{
		{
			name: "interface",
			evidence: LearnedMACEvidence{
				PortID:    17825793,
				Interface: "epon 0/1/2",
				PONNo:     2,
				VLAN:      3501,
			},
		},
		{
			name: "vlan",
			evidence: LearnedMACEvidence{
				PortID:    17825793,
				Interface: "epon 0/1/1",
				PONNo:     1,
				VLAN:      999,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(
					`{"code":0,"description":"success.","data":[` +
						`{"MacAddress":"04:95:E6:58:8E:E8",` +
						`"PortIndex":"epon 0/1/1","OnuId":1,` +
						`"Vlan":3501,"MacType":1}` +
						`]}`,
				))
			}))
			defer server.Close()

			client, err := NewClientWithBaseURL(
				server.URL,
				"user",
				"password",
				server.Client(),
			)
			if err != nil {
				t.Fatal(err)
			}

			resolution, err := client.FindONUByMACWithEvidence(
				context.Background(),
				"session-token",
				"04:95:E6:58:8E:E8",
				&tt.evidence,
			)
			if err == nil {
				t.Fatalf(
					"resolution = %#v, want SNMP/HTTP mismatch error",
					resolution,
				)
			}
		})
	}
}
