package mikrotik

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func newPPPSecretTestClient(
	input string,
) (*client, *bytes.Buffer) {
	reader := bufio.NewReader(
		strings.NewReader(input),
	)

	var output bytes.Buffer

	return &client{
		reader: reader,
		writer: bufio.NewWriter(&output),
	}, &output
}

func TestValidatePPPSecretInput(t *testing.T) {
	valid := PPPSecretInput{
		Name:     "subscriber-1",
		Password: "secret",
		Service:  "pppoe",
		Profile:  "Package-30M",
	}

	if err := validatePPPSecretInput(valid); err != nil {
		t.Fatalf(
			"valid PPP secret input rejected: %v",
			err,
		)
	}

	tests := []struct {
		name  string
		input PPPSecretInput
	}{
		{
			name: "missing name",
			input: PPPSecretInput{
				Password: "secret",
				Profile:  "Package-30M",
			},
		},
		{
			name: "missing password",
			input: PPPSecretInput{
				Name:    "subscriber-1",
				Profile: "Package-30M",
			},
		},
		{
			name: "missing profile",
			input: PPPSecretInput{
				Name:     "subscriber-1",
				Password: "secret",
			},
		},
		{
			name: "unsupported service",
			input: PPPSecretInput{
				Name:     "subscriber-1",
				Password: "secret",
				Service:  "pptp",
				Profile:  "Package-30M",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validatePPPSecretInput(
				test.input,
			); err == nil {
				t.Fatal(
					"expected validation failure",
				)
			}
		})
	}
}

func TestNormalizedPPPServiceDefaultsToPPPoE(
	t *testing.T,
) {
	if got := normalizedPPPService("   "); got != "pppoe" {
		t.Fatalf(
			"default service = %q, want pppoe",
			got,
		)
	}

	if got := normalizedPPPService(
		"PPPOE",
	); got != "pppoe" {
		t.Fatalf(
			"normalized service = %q, want pppoe",
			got,
		)
	}
}

func TestCommandWordsRejectsEmptyCommand(
	t *testing.T,
) {
	c := &client{}

	if _, _, err := c.commandWords(); err == nil {
		t.Fatal(
			"expected empty RouterOS command to fail",
		)
	}
}

func TestSentenceAttributesPreservesDotID(
	t *testing.T,
) {
	attributes := sentenceAttributes(
		[]string{
			"=.id=*A",
			"=name=subscriber-1",
		},
	)

	if attributes[".id"] != "*A" {
		t.Fatalf(
			"unexpected .id %q",
			attributes[".id"],
		)
	}
}

func TestExchangeAcceptsEmptyReply(
	t *testing.T,
) {
	var response bytes.Buffer

	responseWriter := bufio.NewWriter(&response)

	if err := writeSentence(
		responseWriter,
		[]string{"!empty"},
	); err != nil {
		t.Fatalf(
			"prepare !empty response: %v",
			err,
		)
	}

	var request bytes.Buffer

	c := &client{
		reader: bufio.NewReader(&response),
		writer: bufio.NewWriter(&request),
	}

	rows, done, err := c.commandWords(
		"/ppp/secret/set",
		"=.id=*A",
		"=disabled=yes",
	)
	if err != nil {
		t.Fatalf(
			"!empty mutation reply rejected: %v",
			err,
		)
	}

	if len(rows) != 0 {
		t.Fatalf(
			"unexpected rows: %#v",
			rows,
		)
	}

	if len(done) != 0 {
		t.Fatalf(
			"unexpected !empty attributes: %#v",
			done,
		)
	}
}

func decodeTestSentence(
	t *testing.T,
	raw []byte,
) []string {
	t.Helper()

	sentence, err := readSentence(
		bufio.NewReader(
			bytes.NewReader(raw),
		),
	)
	if err != nil {
		t.Fatalf(
			"decode RouterOS sentence: %v",
			err,
		)
	}

	return sentence
}

func TestAddPPPSecretEncodesExpectedSentence(
	t *testing.T,
) {
	var response bytes.Buffer

	responseWriter := bufio.NewWriter(&response)

	if err := writeSentence(
		responseWriter,
		[]string{
			"!done",
			"=ret=*A",
		},
	); err != nil {
		t.Fatalf(
			"prepare add response: %v",
			err,
		)
	}

	var request bytes.Buffer

	c := &client{
		reader: bufio.NewReader(&response),
		writer: bufio.NewWriter(&request),
	}

	id, err := c.addPPPSecret(
		PPPSecretInput{
			Name:          " subscriber-1 ",
			Password:      "subscriber-secret",
			Service:       "PPPOE",
			Profile:       " Package-30M ",
			CallerID:      " C0:A4:76:F7:F7:DD ",
			RemoteAddress: " 10.9.0.220 ",
			Disabled:      false,
		},
	)
	if err != nil {
		t.Fatalf(
			"add PPP secret: %v",
			err,
		)
	}

	if id != "*A" {
		t.Fatalf(
			"returned id = %q, want *A",
			id,
		)
	}

	actual := decodeTestSentence(
		t,
		request.Bytes(),
	)

	expected := []string{
		"/ppp/secret/add",
		"=name=subscriber-1",
		"=password=subscriber-secret",
		"=service=pppoe",
		"=profile=Package-30M",
		"=caller-id=C0:A4:76:F7:F7:DD",
		"=remote-address=10.9.0.220",
		"=disabled=no",
	}

	if len(actual) != len(expected) {
		t.Fatalf(
			"sentence length = %d, want %d; %#v",
			len(actual),
			len(expected),
			actual,
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"word %d = %q, want %q",
				index,
				actual[index],
				expected[index],
			)
		}
	}
}

func TestSetPPPSecretEncodesExpectedSentence(
	t *testing.T,
) {
	var response bytes.Buffer

	responseWriter := bufio.NewWriter(&response)

	if err := writeSentence(
		responseWriter,
		[]string{"!empty"},
	); err != nil {
		t.Fatalf(
			"prepare set response: %v",
			err,
		)
	}

	var request bytes.Buffer

	c := &client{
		reader: bufio.NewReader(&response),
		writer: bufio.NewWriter(&request),
	}

	err := c.setPPPSecret(
		"*B",
		PPPSecretInput{
			Name:     "subscriber-2",
			Password: "replacement-secret",
			Service:  "pppoe",
			Profile:  "Package-50M",
			Disabled: true,
		},
	)
	if err != nil {
		t.Fatalf(
			"set PPP secret: %v",
			err,
		)
	}

	actual := decodeTestSentence(
		t,
		request.Bytes(),
	)

	expected := []string{
		"/ppp/secret/set",
		"=.id=*B",
		"=name=subscriber-2",
		"=password=replacement-secret",
		"=service=pppoe",
		"=profile=Package-50M",
		"=disabled=yes",
	}

	if len(actual) != len(expected) {
		t.Fatalf(
			"sentence length = %d, want %d; %#v",
			len(actual),
			len(expected),
			actual,
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"word %d = %q, want %q",
				index,
				actual[index],
				expected[index],
			)
		}
	}
}

func TestListPPPSecretsFiltersByNameAfterUnfilteredRead(
	t *testing.T,
) {
	var response bytes.Buffer

	responseWriter := bufio.NewWriter(&response)

	if err := writeSentence(
		responseWriter,
		[]string{
			"!re",
			"=.id=*C",
			"=name=subscriber-3",
			"=service=pppoe",
			"=profile=Package-70M",
			"=caller-id=C0:A4:76:F7:F7:DD",
			"=remote-address=10.9.0.220",
			"=disabled=false",
		},
	); err != nil {
		t.Fatalf(
			"prepare list row: %v",
			err,
		)
	}

	if err := writeSentence(
		responseWriter,
		[]string{
			"!re",
			"=.id=*D",
			"=name=another-subscriber",
			"=service=pppoe",
			"=profile=Package-70M",
			"=disabled=false",
		},
	); err != nil {
		t.Fatalf("prepare non-matching list row: %v", err)
	}

	if err := writeSentence(
		responseWriter,
		[]string{"!done"},
	); err != nil {
		t.Fatalf(
			"prepare list done: %v",
			err,
		)
	}

	var request bytes.Buffer

	c := &client{
		reader: bufio.NewReader(&response),
		writer: bufio.NewWriter(&request),
	}

	rows, err := c.listPPPSecrets(
		" subscriber-3 ",
	)
	if err != nil {
		t.Fatalf(
			"list PPP secrets: %v",
			err,
		)
	}

	if len(rows) != 1 {
		t.Fatalf(
			"row count = %d, want 1",
			len(rows),
		)
	}

	if rows[0].ID != "*C" ||
		rows[0].Name != "subscriber-3" ||
		rows[0].Profile != "Package-70M" ||
		rows[0].CallerID != "C0:A4:76:F7:F7:DD" ||
		rows[0].RemoteAddress != "10.9.0.220" ||
		rows[0].Disabled {
		t.Fatalf(
			"unexpected PPP secret: %#v",
			rows[0],
		)
	}

	actual := decodeTestSentence(
		t,
		request.Bytes(),
	)

	expected := []string{
		"/ppp/secret/print",
		"=.proplist=.id,name,service,profile,caller-id,remote-address,disabled",
	}

	if len(actual) != len(expected) {
		t.Fatalf(
			"sentence length = %d, want %d; %#v",
			len(actual),
			len(expected),
			actual,
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"word %d = %q, want %q",
				index,
				actual[index],
				expected[index],
			)
		}
	}
}

func TestPPPSecretStateAndRemoveUseInternalID(
	t *testing.T,
) {
	tests := []struct {
		name     string
		call     func(*client) error
		expected []string
	}{
		{
			name: "disable",
			call: func(c *client) error {
				return c.disablePPPSecret("*D")
			},
			expected: []string{
				"/ppp/secret/set",
				"=.id=*D",
				"=disabled=yes",
			},
		},
		{
			name: "enable",
			call: func(c *client) error {
				return c.enablePPPSecret("*E")
			},
			expected: []string{
				"/ppp/secret/set",
				"=.id=*E",
				"=disabled=no",
			},
		},
		{
			name: "remove",
			call: func(c *client) error {
				return c.removePPPSecret("*F")
			},
			expected: []string{
				"/ppp/secret/remove",
				"=.id=*F",
			},
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				var response bytes.Buffer

				responseWriter :=
					bufio.NewWriter(&response)

				if err := writeSentence(
					responseWriter,
					[]string{"!empty"},
				); err != nil {
					t.Fatalf(
						"prepare mutation response: %v",
						err,
					)
				}

				var request bytes.Buffer

				c := &client{
					reader: bufio.NewReader(
						&response,
					),
					writer: bufio.NewWriter(
						&request,
					),
				}

				if err := test.call(c); err != nil {
					t.Fatalf(
						"mutation failed: %v",
						err,
					)
				}

				actual := decodeTestSentence(
					t,
					request.Bytes(),
				)

				if len(actual) != len(
					test.expected,
				) {
					t.Fatalf(
						"sentence = %#v, want %#v",
						actual,
						test.expected,
					)
				}

				for index := range test.expected {
					if actual[index] !=
						test.expected[index] {
						t.Fatalf(
							"word %d = %q, want %q",
							index,
							actual[index],
							test.expected[index],
						)
					}
				}
			},
		)
	}
}

func TestDisconnectPPPActiveSessionsFindsAndRemovesMatchingUsername(
	t *testing.T,
) {
	var response bytes.Buffer
	responseWriter := bufio.NewWriter(&response)
	for _, sentence := range [][]string{
		{"!re", "=.id=*A", "=name=subscriber-1"},
		{"!re", "=.id=*B", "=name=another-user"},
		{"!done"},
		{"!done"},
	} {
		if err := writeSentence(responseWriter, sentence); err != nil {
			t.Fatalf("prepare RouterOS response: %v", err)
		}
	}
	if err := responseWriter.Flush(); err != nil {
		t.Fatalf("flush RouterOS response: %v", err)
	}

	var request bytes.Buffer
	c := &client{reader: bufio.NewReader(&response), writer: bufio.NewWriter(&request)}
	if err := c.disconnectPPPActiveSessions("subscriber-1"); err != nil {
		t.Fatalf("disconnect active PPP session: %v", err)
	}

	requestReader := bufio.NewReader(bytes.NewReader(request.Bytes()))
	printSentence, err := readSentence(requestReader)
	if err != nil {
		t.Fatalf("decode active-session lookup: %v", err)
	}
	if len(printSentence) != 2 || printSentence[0] != "/ppp/active/print" {
		t.Fatalf("lookup sentence = %#v", printSentence)
	}
	removeSentence, err := readSentence(requestReader)
	if err != nil {
		t.Fatalf("decode active-session removal: %v", err)
	}
	if len(removeSentence) != 2 || removeSentence[0] != "/ppp/active/remove" || removeSentence[1] != "=.id=*A" {
		t.Fatalf("removal sentence = %#v", removeSentence)
	}
}
