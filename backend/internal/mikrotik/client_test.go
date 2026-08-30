package mikrotik

import (
	"bufio"
	"bytes"
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
	if got := parseRouterOSRate("12.5Mbps"); got != 12_500_000 { t.Fatalf("rate = %d", got) }
	if got := parseRouterOSRate("800kbps"); got != 800_000 { t.Fatalf("rate = %d", got) }
	if got := parseRouterOSRate("invalid"); got != 0 { t.Fatalf("invalid rate = %d", got) }
	if got := parseRouterOSBits("3460000"); got != 3_460_000 { t.Fatalf("bits = %d", got) }
	if got := parseRouterOSBits("135kbps"); got != 135_000 { t.Fatalf("bits rate = %d", got) }
	if got := normalizeInterfaceName("  PPPoE-Test_User "); got != "pppoe-test_user" { t.Fatalf("interface = %q", got) }
	if got := parseRouterOSCounterPair("12345/67890"); got != [2]int64{12345, 67890} { t.Fatalf("bytes = %#v", got) }
}
