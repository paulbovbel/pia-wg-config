package pia

import (
	"strings"
	"testing"
)

func testClient(region string, servers []Server, portForward bool) *PIAClient {
	return &PIAClient{
		region:             region,
		wireguardServers:   ServerList{Region(region): servers},
		portForwardRegions: map[Region]bool{Region(region): portForward},
	}
}

func TestWireguardServersForRegionReturnsEveryServer(t *testing.T) {
	servers := []Server{
		{Cn: "first", IP: "10.0.0.1"},
		{Cn: "second", IP: "10.0.0.2"},
		{Cn: "third", IP: "10.0.0.3"},
	}
	client := testClient("ca_toronto", servers, true)

	got, err := client.wireguardServersForRegion()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(servers) {
		t.Fatalf("got %d servers, want %d", len(got), len(servers))
	}

	seen := map[string]bool{}
	for _, server := range got {
		seen[server.Cn] = true
	}
	for _, server := range servers {
		if !seen[server.Cn] {
			t.Fatalf("server %q missing from selection", server.Cn)
		}
	}
}

// Registration must not be pinned to one server: a broken server at the head
// of PIA's list would otherwise poison every attempt until PIA reshuffles.
func TestWireguardServersForRegionDoesNotAlwaysStartWithTheSameServer(t *testing.T) {
	servers := make([]Server, 0, 10)
	for _, cn := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		servers = append(servers, Server{Cn: cn})
	}
	client := testClient("ca_toronto", servers, true)

	first := map[string]bool{}
	for range 50 {
		got, err := client.wireguardServersForRegion()
		if err != nil {
			t.Fatal(err)
		}
		first[got[0].Cn] = true
	}
	if len(first) < 2 {
		t.Fatalf("selection always started with the same server: %v", first)
	}
}

// The original implementation called log.Fatalf here, exiting the process.
func TestWireguardServersForRegionErrorsWhenRegionIsEmpty(t *testing.T) {
	client := testClient("ca_toronto", nil, true)

	if _, err := client.wireguardServersForRegion(); err == nil {
		t.Fatal("expected an error for a region with no servers")
	}
}

func TestRequirePortForwarding(t *testing.T) {
	if err := testClient("ca_toronto", nil, true).RequirePortForwarding(); err != nil {
		t.Fatalf("port forwarding region rejected: %v", err)
	}

	err := testClient("us_chicago", nil, false).RequirePortForwarding()
	if err == nil {
		t.Fatal("expected an error for a region without port forwarding")
	}
	if !strings.Contains(err.Error(), "us_chicago") {
		t.Fatalf("error does not name the region: %v", err)
	}
}

func TestSelectedServerVip(t *testing.T) {
	client := &PIAClient{selectedServerVip: "10.141.0.1"}
	if client.SelectedServerVip() != "10.141.0.1" {
		t.Fatal("wrong gateway address")
	}
}

func TestGenerateWireguardServerListRecordsPortForwarding(t *testing.T) {
	var list piaServerList
	list.Regions = append(list.Regions, struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Country     string `json:"country"`
		AutoRegion  bool   `json:"auto_region"`
		DNS         string `json:"dns"`
		PortForward bool   `json:"port_forward"`
		Geo         bool   `json:"geo"`
		Servers     struct {
			Meta []Server `json:"meta"`
			Wg   []Server `json:"wg"`
		} `json:"servers"`
	}{ID: "ca_toronto", PortForward: true})

	client := &PIAClient{}
	client.wireguardServers = client.generateWireguardServerList(list)

	if !client.portForwardRegions["ca_toronto"] {
		t.Fatal("port forwarding capability was dropped from the server list")
	}
}
