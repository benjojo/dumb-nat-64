package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/getlantern/netx"
)

func main() {
	tport := flag.Int("transport", 1337,
		"the port that iptables will be redirecting connections to")
	flag.Parse()

	la, _ := net.ResolveTCPAddr("tcp6", fmt.Sprintf("[::]:%d", *tport))
	l, err := net.ListenTCP("tcp6", la)
	if err != nil {
		log.Fatalf("Unable to listen on the transparent port %s",
			err.Error())
	}

	failurecount := 0
	for {
		c, err := l.AcceptTCP()
		if err != nil {
			if failurecount != 50 {
				failurecount++
			} else {
				log.Fatalf("Unable to accept connection! %s", err.Error())
			}
			time.Sleep(time.Millisecond * time.Duration(failurecount*10))
			continue
		}
		failurecount = 0

		go handleConn(c)
	}
}

func handleConn(c *net.TCPConn) {
	// first, let's recover the address
	tc, fd, err := realServerAddress(c)
	defer c.Close()
	defer fd.Close()

	if err != nil {
		log.Printf("Unable to recover address %s", err.Error())
		return
	}

	realv4Addr := net.IP(tc.IP[12:16])

	log.Printf("Connection from %s to %s:%d",
		c.RemoteAddr().String(), realv4Addr.To4().String(), tc.Port)

	ipv4Conn, err := netx.DialTimeout("tcp", net.JoinHostPort(realv4Addr.To4().String(), fmt.Sprint(tc.Port)), time.Second*5)
	if err != nil {
		log.Printf("Failed to connect to final target %s", err.Error())
		return
	}
	defer ipv4Conn.Close()

	netx.BidiCopy(c, ipv4Conn, make([]byte, 32768), make([]byte, 32768))
}
