package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/devbay-io/socket_tester/proxyprotocol"
)

var help = flag.Bool("help", false, "Show help")
var host = ""
var port = 0
var proxyProtocol = false
var sslEnabled = false
var tcpCommand = ""
var customTimeoutMillis = 100
var sslSkipChecks = false

func chkErr(err error) {
	if err != nil {
		log.Fatalf("Error: %s", err.Error())
	}
}

func sendRecvTLSMessage(message string, host string, port int, proxyProtocol bool, customTimeoutMillis int) string {
	addr := fmt.Sprintf("%v:%v", host, port)
	cfg := tls.Config{
		InsecureSkipVerify: sslSkipChecks,
	}
	conn, err := proxyprotocol.Dial("tcp", addr, &cfg, proxyProtocol)
	chkErr(err)
	defer conn.Close()
	// Create a proxyprotocol header or use HeaderProxyFromAddrs() if you
	// have two conn's

	_, err = io.WriteString(conn, fmt.Sprintf("%v\n", message))
	chkErr(err)
	reply := make([]byte, 256)
	n, err := conn.Read(reply)
	chkErr(err)
	return string(reply[:n])
}

func sendRecvMessage(message string, host string, port int, proxyProtocol bool, customTimeoutMillis int) string {

	target, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	chkErr(err)
	conn, err := net.DialTCP("tcp", nil, target)

	chkErr(err)
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(customTimeoutMillis) * time.Millisecond))
	chkErr(err)

	defer conn.Close()

	// Create a proxyprotocol header or use HeaderProxyFromAddrs() if you
	// have two conn's
	if proxyProtocol {
		_, err = proxyprotocol.PrepareProxyProtocolHeader().WriteTo(conn)
		chkErr(err)
	}
	_, err = io.WriteString(conn, fmt.Sprintf("%v\n", message))
	chkErr(err)

	buf := make([]byte, 0, 4096) // big buffer
	tmp := make([]byte, 256)     // using small tmo buffer for demonstrating
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					break
				}
				fmt.Println("read error:", err)
			}
		}
		buf = append(buf, tmp[:n]...)
	}
	if len(buf) == 0 {
		chkErr(fmt.Errorf("message: %v to host %v on port %v returned zero length message, proxy protocol was set to %v", message, host, port, proxyProtocol))
	}
	return string(buf)
}

func main() {
	flag.StringVar(&host, "host", "", "host to connect to")
	flag.IntVar(&port, "port", 0, "port to connect to")
	flag.BoolVar(&proxyProtocol, "proxyProtocol", false, "set if you want to use proxy protocol v2")
	flag.StringVar(&tcpCommand, "message", "", "command to be sent to server")
	flag.IntVar(&customTimeoutMillis, "customTimeoutMillis", 100, "milliseconds before timing out socket client")
	flag.BoolVar(&sslEnabled, "sslEnabled", false, "set if you want to test ssl connection")
	flag.BoolVar(&sslSkipChecks, "skipSSLChecks", false, "set if you want to skip certificates checks")
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if sslEnabled {
		fmt.Print(sendRecvTLSMessage(tcpCommand, host, port, proxyProtocol, customTimeoutMillis))
		return
	}
	fmt.Print(sendRecvMessage(tcpCommand, host, port, proxyProtocol, customTimeoutMillis))
}
