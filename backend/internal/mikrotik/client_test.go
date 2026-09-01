package mikrotik

import (
	"bufio"
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestWordLengthRoundTrip(t *testing.T) {
	for _, length := range []uint32{0, 1, 0x7f, 0x80, 0x3fff, 0x4000, 0x1fffff, 0x200000, 0xfffffff, 0x10000000} {
		var encoded bytes.Buffer
		if err := writeLength(&encoded, length); err != nil {
			t.Fatal(err)
		}
		actual, err := readLength(bufio.NewReader(&encoded))
		if err != nil {
			t.Fatal(err)
		}
		if actual != length {
			t.Fatalf("length %d decoded as %d", length, actual)
		}
	}
}

func TestSentenceAttributes(t *testing.T) {
	attributes := sentenceAttributes([]string{"=name=edge-router", "=version=7.20.1", "not-an-attribute"})
	if attributes["name"] != "edge-router" || attributes["version"] != "7.20.1" {
		t.Fatalf("unexpected attributes: %#v", attributes)
	}
}

func TestParseRouterOSSessionTraffic(t *testing.T) {
	if got := parseRouterOSRate("12.5Mbps"); got != 12_500_000 {
		t.Fatalf("rate = %d", got)
	}
	if got := parseRouterOSRate("800kbps"); got != 800_000 {
		t.Fatalf("rate = %d", got)
	}
	if got := parseRouterOSRate("invalid"); got != 0 {
		t.Fatalf("invalid rate = %d", got)
	}
	if got := parseRouterOSBits("3460000"); got != 3_460_000 {
		t.Fatalf("bits = %d", got)
	}
	if got := parseRouterOSBits("135kbps"); got != 135_000 {
		t.Fatalf("bits rate = %d", got)
	}
	if got := normalizeInterfaceName("  PPPoE-Test_User "); got != "pppoe-test_user" {
		t.Fatalf("interface = %q", got)
	}
	if got := parseRouterOSCounterPair("12345/67890"); got != [2]int64{12345, 67890} {
		t.Fatalf("bytes = %#v", got)
	}
}

func TestNormalizeRouterOSARPIdentity(t *testing.T) {
	ip := strings.TrimSpace(" 192.168.1.25 ")
	if net.ParseIP(ip) == nil {
		t.Fatalf("expected valid IP, got %q", ip)
	}

	mac, err := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.ToUpper(mac.String()); got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("MAC = %q", got)
	}

	if net.ParseIP("not-an-ip") != nil {
		t.Fatal("invalid IP unexpectedly parsed")
	}
}

func TestResolveMACFromARPRows(t *testing.T) {
	tests := []struct {
		name string
		rows []map[string]string
		ip   string
		want string
	}{
		{
			name: "valid complete entry",
			rows: []map[string]string{
				{
					"address":     "192.168.1.25",
					"mac-address": "aa:bb:cc:dd:ee:ff",
					"complete":    "true",
				},
			},
			ip:   "192.168.1.25",
			want: "AA:BB:CC:DD:EE:FF",
		},
		{
			name: "wrong IP ignored",
			rows: []map[string]string{
				{
					"address":     "192.168.1.26",
					"mac-address": "AA:BB:CC:DD:EE:FF",
					"complete":    "true",
				},
			},
			ip: "192.168.1.25",
		},
		{
			name: "incomplete ignored",
			rows: []map[string]string{
				{
					"address":     "192.168.1.25",
					"mac-address": "AA:BB:CC:DD:EE:FF",
					"complete":    "false",
				},
			},
			ip: "192.168.1.25",
		},
		{
			name: "invalid ignored",
			rows: []map[string]string{
				{
					"address":     "192.168.1.25",
					"mac-address": "AA:BB:CC:DD:EE:FF",
					"invalid":     "true",
				},
			},
			ip: "192.168.1.25",
		},
		{
			name: "disabled ignored",
			rows: []map[string]string{
				{
					"address":     "192.168.1.25",
					"mac-address": "AA:BB:CC:DD:EE:FF",
					"disabled":    "true",
				},
			},
			ip: "192.168.1.25",
		},
		{
			name: "malformed MAC ignored",
			rows: []map[string]string{
				{
					"address":     "192.168.1.25",
					"mac-address": "not-a-mac",
					"complete":    "true",
				},
			},
			ip: "192.168.1.25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveMACFromARPRows(tt.rows, tt.ip); got != tt.want {
				t.Fatalf("resolveMACFromARPRows() = %q, want %q", got, tt.want)
			}
		})
	}
}
