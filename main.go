package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"syscall"
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

	file, err := l.File()
	if err != nil {
		log.Fatalf("Unable to get file descriptor for listener: %s", err.Error())
	}

	// set IP_TRANSPARENT on the socket, required for TPROXY to work.
	if err := syscall.SetsockoptInt(int(file.Fd()), syscall.IPPROTO_IP, syscall.IP_TRANSPARENT, 1); err != nil {
		file.Close()
		log.Fatalf("Unable to set IP_TRANSPARENT on listener: %s", err.Error())
	}

	file.Close()

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
	defer c.Close()

	addr := c.LocalAddr().(*net.TCPAddr)
	ip := addr.IP[12:16] // To4() not necessary, already 4 bytes.

	log.Printf("Connection from %s to %s:%d",
		c.RemoteAddr().String(), ip.String(), addr.Port)

	ipv4Conn, err := netx.DialTimeout("tcp", net.JoinHostPort(ip.String(), fmt.Sprint(addr.Port)), time.Second*5)
	if err != nil {
		log.Printf("Failed to connect to final target %s", err.Error())
		return
	}
	defer ipv4Conn.Close()

	netx.BidiCopy(c, ipv4Conn, make([]byte, 32768), make([]byte, 32768))
}
