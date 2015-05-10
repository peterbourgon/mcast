package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var (
		addr      = flag.String("addr", "224.0.0.1:1234", "multicast group address")
		maxsz     = flag.Int("maxsz", 8192, "max datagram size")
		heartbeat = flag.Duration("heartbeat", 5*time.Second, "send heartbeat interval")
		identity  = flag.String("identity", mustHostname(), "heartbeat identity")
	)
	flag.Parse()
	log.SetFlags(0)

	errc := make(chan error)
	go func() {
		log.Printf("listening on %s", *addr)
		errc <- server(*addr, *maxsz, recv)
	}()
	go func() {
		log.Printf("sending %q every %s", *identity, *heartbeat)
		errc <- client(*addr, *identity, *heartbeat)
	}()
	go func() {
		errc <- interrupt()
	}()
	log.Fatal(<-errc)
}

func recv(src *net.UDPAddr, buf []byte) {
	log.Printf("%s: %s", src, buf)
}

func server(address string, maxsz int, h func(*net.UDPAddr, []byte)) error {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}

	l, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer l.Close()

	l.SetReadBuffer(maxsz)
	b := make([]byte, maxsz)
	for {

		n, _, _, src, err := l.ReadMsgUDP(b, nil)
		if err != nil {
			log.Printf("ReadFromUDP: %v", err)
			return err
		}
		h(src, b[:n])
	}
}

func client(address, identity string, interval time.Duration) error {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	for tick := time.Tick(interval); ; <-tick {
		fmt.Fprintf(c, identity+"\n")
	}
}

func interrupt() error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return fmt.Errorf("%s", <-c)
}

func mustHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}
