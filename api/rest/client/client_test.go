package client

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ipfs/ipfs-cluster/api/rest"
	"github.com/ipfs/ipfs-cluster/test"

	libp2p "github.com/libp2p/go-libp2p"
	pnet "github.com/libp2p/go-libp2p-pnet"
	ma "github.com/multiformats/go-multiaddr"
)

func testAPI(t *testing.T) *rest.API {
	//logging.SetDebugLogging()
	apiMAddr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")

	cfg := &rest.Config{}
	cfg.Default()
	cfg.HTTPListenAddr = apiMAddr
	var secret [32]byte
	prot, err := pnet.NewV1ProtectorFromBytes(&secret)
	if err != nil {
		t.Fatal(err)
	}

	h, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(apiMAddr),
		libp2p.PrivateNetwork(prot),
	)
	if err != nil {
		t.Fatal(err)
	}

	rest, err := rest.NewAPIWithHost(cfg, h)
	if err != nil {
		t.Fatal("should be able to create a new Api: ", err)
	}

	rest.SetClient(test.NewMockRPCClient(t))
	return rest
}

func shutdown(a *rest.API) {
	a.Shutdown()
	a.Host().Close()
}

func apiMAddr(a *rest.API) ma.Multiaddr {
	listen, _ := a.HTTPAddress()
	hostPort := strings.Split(listen, ":")

	addr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", hostPort[1]))
	return addr
}

func peerMAddr(a *rest.API) ma.Multiaddr {
	listenAddr := a.Host().Addrs()[0]
	ipfsAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", a.Host().ID().Pretty()))
	return listenAddr.Encapsulate(ipfsAddr)
}

func testClientHTTP(t *testing.T, api *rest.API) *Client {
	cfg := &Config{
		APIAddr:           apiMAddr(api),
		DisableKeepAlives: true,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func testClientLibp2p(t *testing.T, api *rest.API) *Client {
	cfg := &Config{
		PeerAddr:          peerMAddr(api),
		ProtectorKey:      make([]byte, 32),
		DisableKeepAlives: true,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestNewClient(t *testing.T) {
	api := testAPI(t)
	defer shutdown(api)

	c := testClientHTTP(t, api)
	if c.p2p != nil {
		t.Error("should not use a libp2p host")
	}

	c = testClientLibp2p(t, api)
	if c.p2p == nil {
		t.Error("expected a libp2p host")
	}
}

func TestDefaultAddress(t *testing.T) {
	cfg := &Config{
		APIAddr:           nil,
		DisableKeepAlives: true,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.hostname != "127.0.0.1:9094" {
		t.Error("default should be used")
	}

	if c.config.ProxyAddr == nil || c.config.ProxyAddr.String() != "/ip4/127.0.0.1/tcp/9095" {
		t.Error("proxy address was not guessed correctly")
	}
}

func TestMultiaddressPrecedence(t *testing.T) {
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	cfg := &Config{
		APIAddr:           addr,
		Host:              "localhost",
		Port:              "9094",
		DisableKeepAlives: true,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.hostname != "1.2.3.4:1234" {
		t.Error("APIAddr should be used")
	}

	if c.config.ProxyAddr == nil || c.config.ProxyAddr.String() != "/ip4/1.2.3.4/tcp/9095" {
		t.Error("proxy address was not guessed correctly")
	}
}

func TestHostPort(t *testing.T) {
	cfg := &Config{
		APIAddr:           nil,
		Host:              "3.3.1.1",
		Port:              "9094",
		DisableKeepAlives: true,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.hostname != "3.3.1.1:9094" {
		t.Error("Host Port should be used")
	}

	if c.config.ProxyAddr == nil || c.config.ProxyAddr.String() != "/ip4/3.3.1.1/tcp/9095" {
		t.Error("proxy address was not guessed correctly")
	}
}

func TestDNSMultiaddress(t *testing.T) {
	addr2, _ := ma.NewMultiaddr("/dns4/localhost/tcp/1234")
	cfg := &Config{
		APIAddr:           addr2,
		Host:              "localhost",
		Port:              "9094",
		DisableKeepAlives: true,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.hostname != "127.0.0.1:1234" {
		t.Error("bad resolved address")
	}

	if c.config.ProxyAddr == nil || c.config.ProxyAddr.String() != "/ip4/127.0.0.1/tcp/9095" {
		t.Error("proxy address was not guessed correctly")
	}
}

func TestPeerAddress(t *testing.T) {
	addr2, _ := ma.NewMultiaddr("/dns4/localhost/tcp/1234")
	peerAddr, _ := ma.NewMultiaddr("/dns4/localhost/tcp/1234/ipfs/QmP7R7gWEnruNePxmCa9GBa4VmUNexLVnb1v47R8Gyo3LP")
	cfg := &Config{
		APIAddr:           addr2,
		Host:              "localhost",
		Port:              "9094",
		DisableKeepAlives: true,
		PeerAddr:          peerAddr,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.hostname != "QmP7R7gWEnruNePxmCa9GBa4VmUNexLVnb1v47R8Gyo3LP" || c.net != "libp2p" {
		t.Error("bad resolved address")
	}

	if c.config.ProxyAddr == nil || c.config.ProxyAddr.String() != "/ip4/127.0.0.1/tcp/9095" {
		t.Error("proxy address was not guessed correctly")
	}
}

func TestProxyAddress(t *testing.T) {
	addr, _ := ma.NewMultiaddr("/ip4/1.3.4.5/tcp/1234")
	cfg := &Config{
		DisableKeepAlives: true,
		ProxyAddr:         addr,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if c.config.ProxyAddr.String() != addr.String() {
		t.Error("proxy address was replaced")
	}
}

func TestIPFS(t *testing.T) {
	ipfsMock := test.NewIpfsMock()
	defer ipfsMock.Close()

	proxyAddr, err := ma.NewMultiaddr(
		fmt.Sprintf("/ip4/%s/tcp/%d", ipfsMock.Addr, ipfsMock.Port),
	)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		DisableKeepAlives: true,
		ProxyAddr:         proxyAddr,
	}

	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ipfs := c.IPFS()

	err = ipfs.Pin(test.TestCid1)
	if err != nil {
		t.Error(err)
	}

	pins, err := ipfs.Pins()
	if err != nil {
		t.Error(err)
	}

	pin, ok := pins[test.TestCid1]
	if !ok {
		t.Error("pin should be in pin list")
	}
	if pin.Type != "recursive" {
		t.Error("pin type unexpected")
	}
}
