package services

import "testing"

func TestParseCustomerChangeRequestedValue(t *testing.T) {
	tests := []struct {
		name        string
		requestType string
		raw         string
		wantError   bool
	}{
		{name: "billing day", requestType: "BILLING_CYCLE", raw: `{"billing_day":15}`},
		{name: "invalid billing day", requestType: "BILLING_CYCLE", raw: `{"billing_day":32}`, wantError: true},
		{name: "package", requestType: "PACKAGE", raw: `{"package_id":7}`},
		{name: "missing package", requestType: "PACKAGE", raw: `{}`, wantError: true},
		{name: "router", requestType: "LINE_SHIFT", raw: `{"router_id":9}`},
		{name: "missing router", requestType: "LINE_SHIFT", raw: `{}`, wantError: true},
		{name: "close ignores value", requestType: "CLOSE", raw: ""},
		{name: "invalid JSON", requestType: "PACKAGE", raw: "7", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseCustomerChangeRequestedValue(test.requestType, test.raw)
			if (err != nil) != test.wantError {
				t.Fatalf("parseCustomerChangeRequestedValue() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}
