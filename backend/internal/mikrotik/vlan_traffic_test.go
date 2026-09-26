package mikrotik

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
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
		if err := reply(
			"!re", "=name=vlan400", "=interface=ether1", "=vlan-id=400", "=disabled=false",
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
			"/interface/monitor-traffic", "=interface=ether1", "=once=",
		}) {
			serverErr <- errors.New("unexpected parent interface traffic request")
			return
		}
		if err := reply("!re", "=name=ether1", "=rx-bits-per-second=1.2Mbps", "=tx-bits-per-second=3.4Mbps"); err != nil {
			serverErr <- err
			return
		}
		if err := reply("!done"); err != nil {
			serverErr <- err
			return
		}
		for _, requestName := range []string{"vlan100", "vlan400"} {
			request, err = readSentence(reader)
			if err != nil {
				serverErr <- err
				return
			}
			if !reflect.DeepEqual(request, []string{
				"/interface/monitor-traffic", "=interface=" + requestName, "=once=",
			}) {
				serverErr <- errors.New("unexpected VLAN interface traffic request")
				return
			}
			if requestName == "vlan100" {
				if err := reply("!re", "=name=vlan100", "=rx-bits-per-second=2.5Mbps", "=tx-bits-per-second=800kbps"); err != nil {
					serverErr <- err
					return
				}
			}
			if err := reply("!done"); err != nil {
				serverErr <- err
				return
			}
		}
		serverErr <- nil
	}()

	address := listener.Addr().(*net.TCPAddr)
	got, err := FetchInterfaceTraffic("127.0.0.1", address.Port, false, "api", "secret", "ether1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.VLANs) != 2 {
		t.Fatalf("VLAN rows = %d, want 2: %#v", len(got.VLANs), got.VLANs)
	}
	want := VLANInterfaceTraffic{
		ParentInterface: "ether1",
		Name:            "vlan100",
		VLANID:          100,
		RxBps:           2_500_000,
		TxBps:           800_000,
	}
	if got.VLANs[0] != want {
		t.Fatalf("VLAN traffic = %#v, want %#v", got.VLANs[0], want)
	}
	wantWithoutTrafficSample := VLANInterfaceTraffic{
		ParentInterface: "ether1",
		Name:            "vlan400",
		VLANID:          400,
	}
	if got.VLANs[1] != wantWithoutTrafficSample {
		t.Fatalf("VLAN without traffic sample = %#v, want %#v", got.VLANs[1], wantWithoutTrafficSample)
	}
	if !reflect.DeepEqual(got.Interfaces, []RouterInterfaceTraffic{
		{Name: "ether1", RxBps: 1_200_000, TxBps: 3_400_000},
		{Name: "vlan100", RxBps: 2_500_000, TxBps: 800_000},
	}) {
		t.Fatalf("interface traffic = %#v", got.Interfaces)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestFetchInterfaceTrafficForInterfaceWithoutVLAN(t *testing.T) {
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
		if len(request) != 3 || request[0] != "/login" || request[1] != "=name=api" {
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
			"/interface/monitor-traffic", "=interface=05.ether1-BDIX-500M", "=once=",
		}) {
			serverErr <- errors.New("traffic request did not target the selected VLAN-free interface")
			return
		}
		if err := reply(
			"!re",
			"=name=05.ether1-BDIX-500M",
			"=rx-bits-per-second=500000",
			"=tx-bits-per-second=750000",
		); err != nil {
			serverErr <- err
			return
		}
		serverErr <- reply("!done")
	}()

	address := listener.Addr().(*net.TCPAddr)
	got, err := FetchInterfaceTraffic("127.0.0.1", address.Port, false, "api", "secret", "05.ether1-BDIX-500M")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Interfaces, []RouterInterfaceTraffic{
		{Name: "05.ether1-BDIX-500M", RxBps: 500_000, TxBps: 750_000},
	}) {
		t.Fatalf("interface traffic = %#v", got.Interfaces)
	}
	if len(got.VLANs) != 0 {
		t.Fatalf("VLAN rows = %#v, want none", got.VLANs)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestMapActivePPPoETrafficMatchesDynamicInterfaces(t *testing.T) {
	activeRows := []map[string]string{
		{"name": "customer-one", "rx-rate": "1Mbps", "tx-rate": "4Mbps"},
		{"name": "customer-two", "interface": "pppoe-customer-two", "rx-rate": "250kbps", "tx-rate": "3Mbps"},
		{"name": "not-sampled", "rx-rate": "12kbps", "tx-rate": "45kbps"},
	}
	trafficRows := []map[string]string{
		{"name": "<pppoe-customer-one>", "rx-bits-per-second": "2Mbps", "tx-bits-per-second": "8Mbps"},
		{"name": "pppoe-customer-two", "rx-bits-per-second": "500kbps", "tx-bits-per-second": "3.5Mbps"},
	}

	got := mapActivePPPoETraffic(activeRows, trafficRows)
	want := []PPPoESessionTraffic{
		{Username: "customer-one", DownloadBps: 8_000_000, UploadBps: 2_000_000, Source: "interface-monitor"},
		{Username: "customer-two", DownloadBps: 3_500_000, UploadBps: 500_000, Source: "interface-monitor"},
		{Username: "not-sampled", DownloadBps: 45_000, UploadBps: 12_000, Source: "ppp-active-rate"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mapped active PPPoE traffic = %#v, want %#v", got, want)
	}
}

func TestMapActivePPPoETrafficFallsBackToActiveRate(t *testing.T) {
	activeRows := []map[string]string{{
		"name":    "customer-one",
		"rx-rate": "250kbps",
		"tx-rate": "2Mbps",
	}}

	got := mapActivePPPoETraffic(activeRows, nil)
	want := []PPPoESessionTraffic{{
		Username:    "customer-one",
		DownloadBps: 2_000_000,
		UploadBps:   250_000,
		Source:      "ppp-active-rate",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback PPPoE traffic = %#v, want %#v", got, want)
	}
}

func TestMonitorActivePPPoEInterfacesBatchesAtRouterOSLimit(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	activeRows := make([]map[string]string, 101)
	for i := range activeRows {
		activeRows[i] = map[string]string{
			"name":      fmt.Sprintf("customer-%03d", i),
			"interface": fmt.Sprintf("<pppoe-customer-%03d>", i),
		}
	}

	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer connection.Close()
		_ = connection.SetDeadline(time.Now().Add(5 * time.Second))

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
		if len(request) != 3 || request[0] != "/login" ||
			request[1] != "=name=api" || request[2] != "=password=secret" {
			serverErr <- errors.New("unexpected RouterOS login request")
			return
		}
		if err := reply("!done"); err != nil {
			serverErr <- err
			return
		}

		for start := 0; start < len(activeRows); start += maxInterfacesPerTrafficSample {
			end := min(start+maxInterfacesPerTrafficSample, len(activeRows))
			request, err = readSentence(reader)
			if err != nil {
				serverErr <- err
				return
			}
			if len(request) != 3 || request[0] != "/interface/monitor-traffic" ||
				request[2] != "=once=" {
				serverErr <- fmt.Errorf("unexpected traffic request: %#v", request)
				return
			}
			names := strings.Split(strings.TrimPrefix(request[1], "=interface="), ",")
			if !strings.HasPrefix(request[1], "=interface=") || len(names) != end-start ||
				len(names) > maxInterfacesPerTrafficSample {
				serverErr <- fmt.Errorf("unexpected interface batch: %#v", names)
				return
			}
			for _, name := range names {
				if err := reply(
					"!re", "=name="+name,
					"=rx-bits-per-second=250kbps", "=tx-bits-per-second=2Mbps",
				); err != nil {
					serverErr <- err
					return
				}
			}
			if err := reply("!done"); err != nil {
				serverErr <- err
				return
			}
		}
		serverErr <- nil
	}()

	address := listener.Addr().(*net.TCPAddr)
	var trafficRows []map[string]string
	var monitored bool
	err = withAuthenticatedClient("127.0.0.1", address.Port, false, "api", "secret", func(c *client) error {
		var monitorErr error
		trafficRows, monitored, monitorErr = monitorActivePPPoEInterfaces(c, activeRows)
		return monitorErr
	})
	if err != nil {
		select {
		case mockErr := <-serverErr:
			t.Fatalf("monitor interfaces: %v (mock: %v)", err, mockErr)
		default:
			t.Fatal(err)
		}
	}
	if !monitored {
		t.Fatal("monitoring did not report a complete sample")
	}
	if len(trafficRows) != len(activeRows) {
		t.Fatalf("traffic rows = %d, want %d", len(trafficRows), len(activeRows))
	}
	traffic := mapActivePPPoETraffic(activeRows, trafficRows)
	if len(traffic) != len(activeRows) {
		t.Fatalf("mapped traffic rows = %d, want %d", len(traffic), len(activeRows))
	}
	for _, sample := range traffic {
		if sample.Source != "interface-monitor" || sample.DownloadBps != 2_000_000 || sample.UploadBps != 250_000 {
			t.Fatalf("unexpected mapped traffic sample: %#v", sample)
		}
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestFetchInterfaceNames(t *testing.T) {
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
		if len(request) != 3 ||
			request[0] != "/login" ||
			request[1] != "=name=api" ||
			!strings.HasPrefix(request[2], "=password=") {
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
			"/interface/print",
			"=.proplist=name,disabled,dynamic",
		}) {
			serverErr <- errors.New("unexpected RouterOS interface list request")
			return
		}
		for _, row := range [][]string{
			{"!re", "=name=ether1", "=disabled=false", "=dynamic=false"},
			{"!re", "=name=bridge1", "=disabled=false", "=dynamic=false"},
			{"!re", "=name=ether2", "=disabled=true", "=dynamic=false"},
			{"!re", "=name=<pppoe-customer>", "=disabled=false", "=dynamic=true"},
			{"!re", "=name=ether1", "=disabled=false", "=dynamic=false"},
		} {
			if err := reply(row...); err != nil {
				serverErr <- err
				return
			}
		}
		serverErr <- reply("!done")
	}()

	address := listener.Addr().(*net.TCPAddr)
	got, err := FetchInterfaceNames("127.0.0.1", address.Port, false, "api", "secret")
	if err != nil {
		select {
		case mockErr := <-serverErr:
			t.Fatalf("FetchInterfaceNames: %v (mock: %v)", err, mockErr)
		default:
			t.Fatal(err)
		}
	}
	want := []string{"ether1", "bridge1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("interfaces = %#v, want %#v", got, want)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}
