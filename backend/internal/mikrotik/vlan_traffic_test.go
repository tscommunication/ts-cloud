package mikrotik

import (
	"bufio"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestFetchVLANInterfaceTraffic(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()
		_ = connection.SetDeadline(time.Now().Add(3 * time.Second))

		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)
		reply := func(words ...string) error {
			if err := writeSentence(writer, words); err != nil {
				return err
			}
			return writer.Flush()
		}
		request, err := readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}
		if !reflect.DeepEqual(request, []string{"/login", "=name=api", "=password=secret"}) {
			serverErr <- errors.New("unexpected RouterOS login request")
			return
		}
		if err := reply("!done"); err != nil {
			serverErr <- err
			return
		}

		request, err = readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}
		if !reflect.DeepEqual(request, []string{
			"/interface/vlan/print",
			"=.proplist=name,interface,vlan-id,disabled",
		}) {
			serverErr <- errors.New("unexpected VLAN list request")
			return
		}
		if err := reply(
			"!re", "=name=vlan100", "=interface=ether1", "=vlan-id=100", "=disabled=false",
		); err != nil {
			serverErr <- err
			return
		}
		if err := reply(
			"!re", "=name=vlan200", "=interface=ether1", "=vlan-id=200", "=disabled=true",
		); err != nil {
			serverErr <- err
			return
		}
		if err := reply(
			"!re", "=name=vlan300", "=interface=ether2", "=vlan-id=300", "=disabled=false",
		); err != nil {
			serverErr <- err
			return
		}
		if err := reply("!done"); err != nil {
			serverErr <- err
			return
		}

		request, err = readSentence(reader)
		if err != nil {
			serverErr <- err
			return
		}
		if !reflect.DeepEqual(request, []string{
			"/interface/monitor-traffic", "=interface=all", "=once=",
		}) {
			serverErr <- errors.New("unexpected interface traffic request")
			return
		}
		for _, row := range [][]string{
			{"!re", "=name=vlan100", "=rx-bits-per-second=2.5Mbps", "=tx-bits-per-second=800kbps"},
			{"!re", "=name=vlan300", "=rx-bits-per-second=1200", "=tx-bits-per-second=3400"},
		} {
			if err := reply(row...); err != nil {
				serverErr <- err
				return
			}
		}
		serverErr <- reply("!done")
	}()

	address := listener.Addr().(*net.TCPAddr)
	got, err := FetchVLANInterfaceTraffic("127.0.0.1", address.Port, false, "api", "secret", "ether1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("VLAN rows = %d, want 1: %#v", len(got), got)
	}
	want := VLANInterfaceTraffic{
		ParentInterface: "ether1",
		Name:            "vlan100",
		VLANID:          100,
		RxBps:           2_500_000,
		TxBps:           800_000,
	}
	if got[0] != want {
		t.Fatalf("VLAN traffic = %#v, want %#v", got[0], want)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}
