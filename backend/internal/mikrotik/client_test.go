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
