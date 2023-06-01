package proxyprotocol

import (
	"context"
	"crypto/tls"
	"net"
	"strings"

	"github.com/pires/go-proxyproto"
)

func PrepareProxyProtocolHeader() *proxyproto.Header {
	header := &proxyproto.Header{
		Version:           2,
		Command:           proxyproto.PROXY,
		TransportProtocol: proxyproto.TCPv4,
		SourceAddr: &net.TCPAddr{
			IP:   net.ParseIP("10.0.0.0"),
			Port: 1883,
		},
		DestinationAddr: &net.TCPAddr{
			IP:   net.ParseIP("20.0.0.0"),
			Port: 1883,
		},
	}
	return header
}

// DialWithDialer connects to the given network address using dialer.Dial and
// then initiates a TLS handshake, returning the resulting TLS connection. Any
// timeout or deadline given in the dialer apply to connection and TLS
// handshake as a whole.
//
// DialWithDialer interprets a nil configuration as equivalent to the zero
// configuration; see the documentation of Config for the defaults.
//
// DialWithDialer uses context.Background internally; to specify the context,
// use Dialer.DialContext with NetDialer set to the desired dialer.
func DialWithDialer(dialer *net.Dialer, network, addr string, config *tls.Config, proxyProtocol bool) (*tls.Conn, error) {
	return dial(context.Background(), dialer, network, addr, config, proxyProtocol)
}

func dial(ctx context.Context, netDialer *net.Dialer, network, addr string, config *tls.Config, proxyProtocol bool) (*tls.Conn, error) {
	if netDialer.Timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, netDialer.Timeout)
		defer cancel()
	}

	if !netDialer.Deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, netDialer.Deadline)
		defer cancel()
	}

	rawConn, err := netDialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	if proxyProtocol {
		_, err = PrepareProxyProtocolHeader().WriteTo(rawConn)
		if err != nil {
			return nil, err
		}
	}

	colonPos := strings.LastIndex(addr, ":")
	if colonPos == -1 {
		colonPos = len(addr)
	}
	hostname := addr[:colonPos]

	if config == nil {
		config = defaultConfig()
	}
	// If no ServerName is set, infer the ServerName
	// from the hostname we're connecting to.
	if config.ServerName == "" {
		// Make a copy to avoid polluting argument or default.
		c := config.Clone()
		c.ServerName = hostname
		config = c
	}

	conn := tls.Client(rawConn, config)
	if err := conn.HandshakeContext(ctx); err != nil {
		rawConn.Close()
		return nil, err
	}
	return conn, nil
}

// Dial connects to the given network address using net.Dial
// and then initiates a TLS handshake, returning the resulting
// TLS connection.
// Dial interprets a nil configuration as equivalent to
// the zero configuration; see the documentation of Config
// for the defaults.
func Dial(network, addr string, config *tls.Config, proxyProtocol bool) (*tls.Conn, error) {
	return DialWithDialer(new(net.Dialer), network, addr, config, proxyProtocol)
}

var emptyConfig tls.Config

func defaultConfig() *tls.Config {
	return &emptyConfig
}
