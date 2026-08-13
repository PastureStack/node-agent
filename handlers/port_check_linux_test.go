package handlers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const procNetHeader = "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n"

func TestParseProcNetUsesListeningTcpAndUnconnectedUdpSockets(t *testing.T) {
	root, err := ioutil.TempDir("", "port-check-proc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	path := filepath.Join(root, "tcp")
	content := procNetHeader +
		"   0: 0100007F:0899 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 1\n" +
		"   1: 00000000:0899 00000000:0000 01 00000000:00000000 00:00000000 00000000 0 0 2\n"
	if err := ioutil.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	values, err := parseProcNet(path, "tcp", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].BindAddress != "127.0.0.1" || values[0].PublicPort != 2201 {
		t.Fatalf("unexpected TCP socket inventory: %#v", values)
	}

	udpPath := filepath.Join(root, "udp")
	udpContent := procNetHeader +
		"   0: 00000000:0899 00000000:0000 07 00000000:00000000 00:00000000 00000000 0 0 3\n"
	if err := ioutil.WriteFile(udpPath, []byte(udpContent), 0600); err != nil {
		t.Fatal(err)
	}
	values, err = parseProcNet(udpPath, "udp", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].Protocol != "udp" || values[0].BindAddress != "0.0.0.0" {
		t.Fatalf("unexpected UDP socket inventory: %#v", values)
	}
}

func TestDecodeProcIPv6Address(t *testing.T) {
	value, err := decodeProcAddress("00000000000000000000000001000000", true)
	if err != nil {
		t.Fatal(err)
	}
	if value != "::1" {
		t.Fatalf("expected ::1, got %s", value)
	}
}

func TestHostSocketProbeFailsClosedWhenInventoryIsIncomplete(t *testing.T) {
	root, err := ioutil.TempDir("", "port-check-host-proc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	netRoot := filepath.Join(root, "net")
	if err := os.MkdirAll(netRoot, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tcp", "tcp6", "udp", "udp6"} {
		content := procNetHeader
		if name == "tcp" {
			content += "   0: 00000000:0899 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 1\n"
		}
		if err := ioutil.WriteFile(filepath.Join(netRoot, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	requested := []portCheckPort{{BindAddress: "0.0.0.0", PublicPort: 2201, Protocol: "tcp"}}
	conflicts, supported := hostSocketConflictsAt(root, requested, map[string]bool{})
	if !supported || len(conflicts) != 1 || conflicts[0].Source != "hostProcess" {
		t.Fatalf("complete host inventory was not reported: supported=%v conflicts=%#v", supported, conflicts)
	}

	if err := os.Remove(filepath.Join(netRoot, "udp6")); err != nil {
		t.Fatal(err)
	}
	_, supported = hostSocketConflictsAt(root, requested, map[string]bool{})
	if supported {
		t.Fatal("a partial /proc/net inventory must not be reported as complete")
	}
}
