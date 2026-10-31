package main

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

var (
	errPrivateAddress = errors.New("refusing a non-public address")
	sharedNAT         = netip.MustParsePrefix("100.64.0.0/10")
)

// publicAddress tells if an address can be reached from the internet.
func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !sharedNAT.Contains(ip)
}

// refusePrivate runs after DNS resolution, so a public name that points inside
// the network is caught too, and on every redirect.
func refusePrivate(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil || !publicAddress(ap.Addr()) {
		return errPrivateAddress
	}
	return nil
}

// mediaClient fetches addresses that come from posts and profiles, which remote
// servers influence: only https, only public addresses, few redirects.
func mediaClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{Timeout: 5 * time.Second, Control: refusePrivate}).DialContext
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" || len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}
