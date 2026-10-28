package pia

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/benburkert/dns"
	"github.com/pkg/errors"
)

type PIAWgClient interface {
	GetToken() (string, error)
	AddKey(token, publickey string) (AddKeyResult, error)
}

type Region string
type ServerList map[Region][]Server

type PIAClient struct {
	region             string
	wireguardServers   ServerList
	username           string
	password           string
	verbose            bool
	caCert             []byte
	selectedServerName string
	selectedServerVip  string
	portForwardRegions map[Region]bool
}

// SelectedServerName is the TLS name of the server whose key registration
// succeeded. Gluetun needs this exact name for PIA port forwarding.
func (p *PIAClient) SelectedServerName() string { return p.selectedServerName }

// SelectedServerVip is the in-tunnel address of the server that accepted the
// key. PIA's port forwarding API listens there, and it is not derivable from
// the peer address, so a consumer that does not know it has to guess.
func (p *PIAClient) SelectedServerVip() string { return p.selectedServerVip }

// RequirePortForwarding reports whether the configured region offers PIA port
// forwarding, so an unusable region fails before any key is registered.
func (p *PIAClient) RequirePortForwarding() error {
	if !p.portForwardRegions[Region(p.region)] {
		return fmt.Errorf("region %q does not offer port forwarding", p.region)
	}
	return nil
}

type piaServerList struct {
	Regions []struct {
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
	} `json:"regions"`
}

type AddKeyResult struct {
	Status     string   `json:"status"`
	ServerKey  string   `json:"server_key"`
	ServerPort int      `json:"server_port"`
	ServerIP   string   `json:"server_ip"`
	ServerVip  string   `json:"server_vip"`
	PeerIP     string   `json:"peer_ip"`
	PeerPubkey string   `json:"peer_pubkey"`
	DNSServers []string `json:"dns_servers"`
}

type Server struct {
	Cn string
	IP string
}

// NewPIAClient creates a new PIA client for with the list of servers populated
func NewPIAClient(username, password, region string, verbose bool) (*PIAClient, error) {
	piaClient := PIAClient{
		username: username,
		password: password,
		region:   region,
		verbose:  verbose,
	}

	// Get list of servers
	serverList, err := piaClient.getServerList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch server list from PIA")
	}

	// Set servers
	piaClient.wireguardServers = piaClient.generateWireguardServerList(serverList)

	// Validate region exists
	if _, exists := piaClient.wireguardServers[Region(region)]; !exists {
		availableRegions := make([]string, 0, len(piaClient.wireguardServers))
		for r := range piaClient.wireguardServers {
			availableRegions = append(availableRegions, string(r))
		}
		return nil, fmt.Errorf("region '%s' not found. Available regions: %v. Use 'pia-wg-config regions' to see all available regions", region, availableRegions[:5]) // Show first 5 as example
	}

	return &piaClient, nil
}

// tokenURL is PIA's central token API (the same endpoint used by PIA's
// manual-connections scripts). Regional metadata servers no longer serve
// /authv3/generateToken reliably (issues #12, #17).
const tokenURL = "https://www.privateinternetaccess.com/api/client/v2/token"

// GetToken
func (p *PIAClient) GetToken() (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(tokenURL, url.Values{
		"username": {p.username},
		"password": {p.password},
	})
	if err != nil {
		return "", errors.Wrap(err, "error executing request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed: %s", resp.Status)
	}
	// Parse response
	var tokenResp struct {
		Token string `json:"token"`
	}

	err = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&tokenResp)
	if err != nil {
		return "", errors.Wrap(err, "error decoding token response")
	}

	if p.verbose {
		log.Print("Got token")
	}

	return tokenResp.Token, nil
}

// GetAvailableRegions returns all available regions
func (p *PIAClient) GetAvailableRegions() (map[Region]string, error) {
	serverList, err := p.getServerList()
	if err != nil {
		return nil, err
	}

	regions := make(map[Region]string)
	for _, r := range serverList.Regions {
		regions[Region(r.ID)] = r.Name
	}

	return regions, nil
}

// AddKey registers publickey with a WireGuard server in the configured region.
// A region lists several servers and any one of them may refuse the key or be
// missing the port forwarding service, so try them until one accepts.
func (p *PIAClient) AddKey(token, publickey string) (AddKeyResult, error) {
	servers, err := p.wireguardServersForRegion()
	if err != nil {
		return AddKeyResult{}, err
	}

	failures := make([]string, 0, len(servers))
	for _, server := range servers {
		result, err := p.addKeyToServer(server, token, publickey)
		if err == nil {
			p.selectedServerName = server.Cn
			p.selectedServerVip = result.ServerVip
			return result, nil
		}
		if p.verbose {
			log.Printf("Server %s did not accept the key: %v", server.Cn, err)
		}
		failures = append(failures, fmt.Sprintf("%s: %v", server.Cn, err))
	}

	return AddKeyResult{}, fmt.Errorf("no WireGuard server in region %q accepted the key: %s",
		p.region, strings.Join(failures, "; "))
}

func (p *PIAClient) addKeyToServer(server Server, token, publickey string) (AddKeyResult, error) {
	var addKeyResp AddKeyResult

	// Build http request
	requestURL := fmt.Sprintf("https://%v:1337/addKey?pt=%v&pubkey=%v", server.Cn, url.QueryEscape(token), url.QueryEscape(publickey))

	// Send request
	resp, err := p.executePIARequest(server, requestURL, token)
	if err != nil {
		return addKeyResp, errors.Wrap(err, "error executing request")
	}
	defer resp.Body.Close()

	// Parse response
	err = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&addKeyResp)
	if err != nil {
		return addKeyResp, errors.Wrap(err, "error decoding add key response")
	}

	// PIA reports refusal in the body rather than the status code.
	if !strings.EqualFold(addKeyResp.Status, "OK") {
		return AddKeyResult{}, fmt.Errorf("status %q", addKeyResp.Status)
	}

	return addKeyResp, nil
}

// wireguardServersForRegion returns the region's WireGuard servers in a random
// order. PIA lists them deterministically, so always taking the first pins
// every registration to one server, including while that server is broken.
func (p *PIAClient) wireguardServersForRegion() ([]Server, error) {
	if p.verbose {
		log.Print("Getting wireguard servers for region: ", p.region)
	}
	servers := p.wireguardServers[Region(p.region)]
	if len(servers) == 0 {
		return nil, fmt.Errorf("no WireGuard servers available for region %q", p.region)
	}

	shuffled := make([]Server, len(servers))
	copy(shuffled, servers)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled, nil
}

// getSeverList returns a list of servers from the PIA API
func (p *PIAClient) getServerList() (piaServerList, error) {
	var serverList piaServerList

	resp, err := http.Get("https://serverlist.piaservers.net/vpninfo/servers/v4")
	if err != nil {
		return piaServerList{}, err
	}

	// Strip the base64 garbage
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return piaServerList{}, err
	}
	respString := string(respBytes)
	lastBracketInd := strings.LastIndex(respString, "}")
	safeJSON := respString[:lastBracketInd+1]

	// Parse the JSON
	err = json.Unmarshal([]byte(safeJSON), &serverList)
	if err != nil {
		return piaServerList{}, err
	}

	// Return list of servers
	return serverList, nil
}

// generateWireguardServerList
func (p *PIAClient) generateWireguardServerList(list piaServerList) ServerList {
	servers := ServerList{}
	p.portForwardRegions = make(map[Region]bool, len(list.Regions))

	for _, r := range list.Regions {
		p.portForwardRegions[Region(r.ID)] = r.PortForward
		for _, server := range r.Servers.Wg {
			servers[Region(r.ID)] = append(servers[Region(r.ID)], Server{
				Cn: server.Cn,
				IP: server.IP,
			})
		}
	}

	return servers
}

func (p *PIAClient) executePIARequest(server Server, requestURL, token string) (*http.Response, error) {
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	// Set header to JSON
	req.Header.Set("Content-Type", "application/json")

	// Set basic auth
	if token == "" {
		req.SetBasicAuth(p.username, p.password)
	}

	// Add certificate to shared pool
	err = p.downloadPIACertificate()
	if err != nil {
		return nil, errors.Wrap(err, "error downloading ca certificate")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(p.caCert)

	// Create DNS resolver for PIA addresses
	zone := &dns.Zone{
		Origin: "",
		TTL:    5 * time.Minute,
		RRs: dns.RRSet{
			server.Cn: {
				dns.TypeA: []dns.Record{
					&dns.A{A: net.ParseIP(server.IP)},
				},
			},
		},
	}
	mux := new(dns.ResolveMux)
	mux.Handle(dns.TypeANY, zone.Origin, zone)
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: (&dns.Client{
			Resolver: mux,
		}).Dial,
	}

	// Set custom DNS server
	dialer := &net.Dialer{
		Resolver: resolver,
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
			DialContext: dialContext,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		// HTTP errors include addKey's token-bearing URL; log only the cause.
		var urlError *url.Error
		if errors.As(err, &urlError) {
			err = urlError.Err
		}
		return nil, err
	}

	// Return error if status code is not 200
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("status code %v", resp.StatusCode)
	}

	return resp, nil
}

// downloadPIACertificate downloads the PIA certificate
func (p *PIAClient) downloadPIACertificate() error {
	// caCert already loaded
	if len(p.caCert) > 0 {
		return nil
	}

	// Download certificate
	resp, err := http.Get("https://raw.githubusercontent.com/pia-foss/manual-connections/645377e87238a52596c004f760d238e3faea68d1/ca.rsa.4096.crt")
	if err != nil {
		return err
	}

	// Parse certificate
	p.caCert, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}
