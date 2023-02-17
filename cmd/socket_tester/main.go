package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/pires/go-proxyproto"
)

var help = flag.Bool("help", false, "Show help")
var host = ""
var port = 0
var proxyProtocol = false
var tcpCommand = ""
var customTimeoutMillis = 100

func chkErr(err error) {
	if err != nil {
		log.Fatalf("Error: %s", err.Error())
	}
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
		_, err = header.WriteTo(conn)
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
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	fmt.Print(sendRecvMessage(tcpCommand, host, port, proxyProtocol, customTimeoutMillis))
}
