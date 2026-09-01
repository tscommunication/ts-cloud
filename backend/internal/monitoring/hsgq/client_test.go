package hsgq

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginAndResolveLearnedMAC(t *testing.T) {
	const (
		username = "root"
		password = "test-secret"
		token    = "test-token"
		target   = "80:AF:CA:AE:6A:B5"
	)

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			switch {
			case r.URL.Path == "/userlogin" &&
				r.URL.Query().Get("form") == "login":
				var payload struct {
					Method string `json:"method"`
					Param  struct {
						Name  string `json:"name"`
						Key   string `json:"key"`
						Value string `json:"value"`
					} `json:"param"`
				}

				if err := json.NewDecoder(
					r.Body,
				).Decode(&payload); err != nil {
					t.Fatalf(
						"decode login payload: %v",
						err,
					)
				}

				expectedHash := md5.Sum(
					[]byte(username + ":" + password),
				)

				if payload.Method != "set" ||
					payload.Param.Name != username ||
					payload.Param.Key !=
						hex.EncodeToString(expectedHash[:]) ||
					payload.Param.Value !=
						base64.StdEncoding.EncodeToString(
							[]byte(password),
						) {
					t.Fatalf(
						"unexpected login payload: %+v",
						payload,
					)
				}

				w.Header().Set(
					"X-Token",
					token,
				)
				_, _ = w.Write(
					[]byte(
						`{"code":1,"message":"success"}`,
					),
				)

			case r.URL.Path == "/pon_mac" &&
				r.URL.Query().Get("form") == "table":
				if r.Header.Get("X-Token") != token {
					t.Fatalf("missing X-Token")
				}
				_, _ = w.Write(
					[]byte(
						`{"code":1,"message":"success"}`,
					),
				)

			case r.URL.Path == "/pon_mac_table":
				if r.Header.Get("X-Token") != token {
					t.Fatalf("missing X-Token")
				}
				_, _ = w.Write([]byte(
					`{"code":1,"message":"success","data":[` +
						`{"macaddr":"80:af:ca:ae:6a:b5",` +
						`"vlan_id":1,"port_id":1,"onu_id":4,` +
						`"mac_type":0,"onu_name":"ONU01/04"}` +
						`]}`,
				))

			default:
				http.NotFound(w, r)
			}
		}),
	)
	defer server.Close()

	client, err := NewClientWithBaseURL(
		server.URL,
		username,
		password,
		server.Client(),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	gotToken, err := client.Login(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if gotToken != token {
		t.Fatalf(
			"token=%q want=%q",
			gotToken,
			token,
		)
	}

	got, err := client.ResolveLearnedMAC(
		context.Background(),
		gotToken,
		target,
	)
	if err != nil {
		t.Fatalf(
			"resolve learned MAC: %v",
			err,
		)
	}
	if got == nil {
		t.Fatal("expected learned MAC match")
	}

	if got.MACAddress != "80:af:ca:ae:6a:b5" ||
		got.VLANID != 1 ||
		got.PONNo != 1 ||
		got.ONUNo != 4 ||
		got.MACType != 0 ||
		got.ONUName != "ONU01/04" {
		t.Fatalf(
			"unexpected resolution: %+v",
			got,
		)
	}
}
