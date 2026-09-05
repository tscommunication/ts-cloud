package mikrotik

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeRouterOSRequest struct {
	Words []string
}

func startFakeRouterOSServer(
	t *testing.T,
	handler func(
		net.Conn,
		[]fakeRouterOSRequest,
	),
	expectedRequests int,
) (string, int) {
	t.Helper()

	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf(
			"listen fake RouterOS server: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = listener.Close()
	})

	address := listener.Addr().(*net.TCPAddr)

	done := make(chan struct{})

	go func() {
		defer close(done)

		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()

		_ = connection.SetDeadline(
			time.Now().Add(3 * time.Second),
		)

		reader := bufio.NewReader(connection)

		requests := make(
			[]fakeRouterOSRequest,
			0,
			expectedRequests,
		)

		for len(requests) < expectedRequests {
			words, err := readSentence(reader)
			if err != nil {
				return
			}

			requests = append(
				requests,
				fakeRouterOSRequest{
					Words: words,
				},
			)
		}

		handler(
			connection,
			requests,
		)
	}()

	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(4 * time.Second):
			t.Error(
				"fake RouterOS server did not stop",
			)
		}
	})

	return "127.0.0.1", address.Port
}

func writeFakeRouterOSReply(
	t *testing.T,
	connection net.Conn,
	words ...string,
) {
	t.Helper()

	writer := bufio.NewWriter(connection)

	if err := writeSentence(
		writer,
		words,
	); err != nil {
		t.Errorf(
			"write fake RouterOS reply: %v",
			err,
		)
	}
}

func expectRouterOSWords(
	t *testing.T,
	actual []string,
	expected []string,
) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf(
			"word count = %d, want %d\nactual=%#v\nexpected=%#v",
			len(actual),
			len(expected),
			actual,
			expected,
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"word %d = %q, want %q\nactual=%#v",
				index,
				actual[index],
				expected[index],
				actual,
			)
		}
	}
}

func TestAddPPPSecretExportedSurfaceAuthenticatesAndMutates(
	t *testing.T,
) {
	const (
		routerUser     = "api-admin"
		routerPassword = "router-api-secret"
		pppoePassword  = "subscriber-secret"
	)

	var listener net.Listener

	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().(*net.TCPAddr)

	serverErr := make(chan error, 1)

	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()

		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)

		loginWords, err := readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}

		expectedLogin := []string{
			"/login",
			"=name=" + routerUser,
			"=password=" + routerPassword,
		}

		if len(loginWords) != len(expectedLogin) {
			serverErr <- &fakeRouterOSAssertionError{
				message: "unexpected login word count",
			}
			return
		}

		for index := range expectedLogin {
			if loginWords[index] != expectedLogin[index] {
				serverErr <- &fakeRouterOSAssertionError{
					message: "unexpected login sentence",
				}
				return
			}
		}

		if err := writeSentence(
			writer,
			[]string{"!done"},
		); err != nil {
			serverErr <- err
			return
		}

		mutationWords, err := readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}

		expectedMutation := []string{
			"/ppp/secret/add",
			"=name=subscriber-1001",
			"=password=" + pppoePassword,
			"=service=pppoe",
			"=profile=PKG-20M",
			"=disabled=no",
		}

		if len(mutationWords) != len(expectedMutation) {
			serverErr <- &fakeRouterOSAssertionError{
				message: "unexpected mutation word count",
			}
			return
		}

		for index := range expectedMutation {
			if mutationWords[index] !=
				expectedMutation[index] {
				serverErr <- &fakeRouterOSAssertionError{
					message: "unexpected mutation sentence",
				}
				return
			}
		}

		if err := writeSentence(
			writer,
			[]string{
				"!done",
				"=ret=*17",
			},
		); err != nil {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	id, err := AddPPPSecret(
		"127.0.0.1",
		address.Port,
		false,
		routerUser,
		routerPassword,
		PPPSecretInput{
			Name:     "subscriber-1001",
			Password: pppoePassword,
			Service:  "pppoe",
			Profile:  "PKG-20M",
			Disabled: false,
		},
	)
	if err != nil {
		t.Fatalf(
			"AddPPPSecret: %v",
			err,
		)
	}

	if id != "*17" {
		t.Fatalf(
			"created PPP secret id = %q, want %q",
			id,
			"*17",
		)
	}

	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

type fakeRouterOSAssertionError struct {
	message string
}

func (err *fakeRouterOSAssertionError) Error() string {
	return err.message
}

func TestListPPPSecretsExportedSurfaceAuthenticatesAndReads(
	t *testing.T,
) {
	const (
		routerUser     = "api-reader"
		routerPassword = "router-secret"
	)

	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().(*net.TCPAddr)

	serverErr := make(chan error, 1)

	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()

		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)

		loginWords, err := readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}

		expectedLogin := []string{
			"/login",
			"=name=" + routerUser,
			"=password=" + routerPassword,
		}

		if strings.Join(loginWords, "\x00") !=
			strings.Join(expectedLogin, "\x00") {
			serverErr <- &fakeRouterOSAssertionError{
				message: "unexpected list login sentence",
			}
			return
		}

		if err := writeSentence(
			writer,
			[]string{"!done"},
		); err != nil {
			serverErr <- err
			return
		}

		listWords, err := readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}

		expectedList := []string{
			"/ppp/secret/print",
			"=.proplist=.id,name,service,profile,caller-id,remote-address,disabled",
		}

		if strings.Join(listWords, "\x00") !=
			strings.Join(expectedList, "\x00") {
			serverErr <- &fakeRouterOSAssertionError{
				message: "unexpected list PPP secret sentence",
			}
			return
		}

		if err := writeSentence(
			writer,
			[]string{
				"!re",
				"=.id=*21",
				"=name=subscriber-1002",
				"=service=pppoe",
				"=profile=PKG-30M",
				"=disabled=false",
			},
		); err != nil {
			serverErr <- err
			return
		}

		if err := writeSentence(
			writer,
			[]string{"!done"},
		); err != nil {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	rows, err := ListPPPSecrets(
		"127.0.0.1",
		address.Port,
		false,
		routerUser,
		routerPassword,
		"subscriber-1002",
	)
	if err != nil {
		t.Fatalf(
			"ListPPPSecrets: %v",
			err,
		)
	}

	if len(rows) != 1 {
		t.Fatalf(
			"PPP secret count = %d, want 1",
			len(rows),
		)
	}

	row := rows[0]

	if row.ID != "*21" {
		t.Fatalf(
			"id = %q, want *21",
			row.ID,
		)
	}

	if row.Name != "subscriber-1002" {
		t.Fatalf(
			"name = %q",
			row.Name,
		)
	}

	if row.Service != "pppoe" {
		t.Fatalf(
			"service = %q",
			row.Service,
		)
	}

	if row.Profile != "PKG-30M" {
		t.Fatalf(
			"profile = %q",
			row.Profile,
		)
	}

	if row.Disabled {
		t.Fatal(
			"expected PPP secret to be enabled",
		)
	}

	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestDisablePPPSecretExportedSurfaceUsesInternalID(
	t *testing.T,
) {
	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().(*net.TCPAddr)

	serverErr := make(chan error, 1)

	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()

		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)

		if _, err := readSentence(reader); err != nil {
			serverErr <- err
			return
		}

		if err := writeSentence(
			writer,
			[]string{"!done"},
		); err != nil {
			serverErr <- err
			return
		}

		words, err := readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}

		expected := []string{
			"/ppp/secret/set",
			"=.id=*55",
			"=disabled=yes",
		}

		if strings.Join(words, "\x00") !=
			strings.Join(expected, "\x00") {
			serverErr <- &fakeRouterOSAssertionError{
				message: "unexpected disable sentence",
			}
			return
		}

		if err := writeSentence(
			writer,
			[]string{"!empty"},
		); err != nil {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	err = DisablePPPSecret(
		"127.0.0.1",
		address.Port,
		false,
		"api-admin",
		"router-secret",
		"*55",
	)
	if err != nil {
		t.Fatalf(
			"DisablePPPSecret: %v",
			err,
		)
	}

	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestExportedPPPSecretSurfaceConnectionFailure(
	t *testing.T,
) {
	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("reserve local port: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	if err := listener.Close(); err != nil {
		t.Fatalf(
			"close reserved listener: %v",
			err,
		)
	}

	_, err = ListPPPSecrets(
		"127.0.0.1",
		port,
		false,
		"api-user",
		"router-secret",
		"",
	)
	if err == nil {
		t.Fatal(
			"expected connection failure",
		)
	}

	var connectionErr *ConnectionError

	if !strings.Contains(
		err.Error(),
		"connect to RouterOS API",
	) {
		t.Fatalf(
			"unexpected connection error: %v",
			err,
		)
	}

	_ = connectionErr
}

func TestWithAuthenticatedClientRejectsNilOperation(
	t *testing.T,
) {
	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().(*net.TCPAddr)

	serverErr := make(chan error, 1)

	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()

		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)

		words, err := readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}

		expected := []string{
			"/login",
			"=name=api-user",
			"=password=router-secret",
		}

		if strings.Join(words, "\x00") !=
			strings.Join(expected, "\x00") {
			serverErr <- &fakeRouterOSAssertionError{
				message: "unexpected nil-operation login sentence",
			}
			return
		}

		if err := writeSentence(
			writer,
			[]string{"!done"},
		); err != nil {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	err = withAuthenticatedClient(
		"127.0.0.1",
		address.Port,
		false,
		"api-user",
		"router-secret",
		nil,
	)
	if err == nil {
		t.Fatal(
			"expected nil operation to fail",
		)
	}

	if err.Error() !=
		"RouterOS operation is required" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestFakeRouterOSPortContract(
	t *testing.T,
) {
	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, rawPort, err := net.SplitHostPort(
		listener.Addr().String(),
	)
	if err != nil {
		t.Fatalf(
			"split host port: %v",
			err,
		)
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 {
		t.Fatalf(
			"invalid local fake RouterOS port %q",
			rawPort,
		)
	}
}
