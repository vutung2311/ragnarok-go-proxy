package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var (
	accountServerAddr = "117.103.206.234:6900"
	localAddr         = "127.0.0.1:6900"
	remoteAddr        = accountServerAddr
)

func proxyConn(lConn *net.TCPConn) {
	rAddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		panic(err)
	}
	log.Println("Remote address:", rAddr.String())

	rConn, err := net.DialTCP("tcp", nil, rAddr)
	if err != nil {
		panic(err)
	}
	defer func() { _ = rConn.Close() }()

	ctx, cancelFn := context.WithCancel(context.Background())
	go pipeConn(ctx, cancelFn, lConn, rConn, false)
	go pipeConn(ctx, cancelFn, rConn, lConn, true)
	for {
		select {
		case <-ctx.Done():
			remoteAddr = accountServerAddr
			return
		default:
			<-time.After(100 * time.Millisecond)
		}
	}
}

func pipeConn(
	ctx context.Context,
	cancelFn context.CancelFunc,
	in *net.TCPConn,
	out *net.TCPConn,
	applyTransform bool,
) {
	defer cancelFn()
	var buf [0xffff]byte
	for {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}

		n, err := in.Read(buf[:])
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("ERR: %+v", err)
			return
		}
		if n == 0 {
			continue
		}
		log.Printf("received from %s:\n%v", in.RemoteAddr(), hex.Dump(buf[:n]))
		finalBuf := buf[:]
		if applyTransform {
			finalBuf, err = transform(buf[:], n)
			if err != nil {
				log.Printf("ERR: %+v", err)
				return
			}
		}
		_, err = out.Write(finalBuf[:n])
		if err != nil {
			log.Printf("ERR: %+v", err)
			return
		}
		log.Printf("sent to %s:\n%v", out.RemoteAddr(), hex.Dump(finalBuf[:n]))
	}
}

func transform(in []byte, size int) ([]byte, error) {
	var m Message
	err := m.Read(in, size)
	if err != nil && !errors.Is(err, io.EOF) {
		panic(err)
	}
	if m.ID == 0xb07 {
		var (
			origIpAddr   = make(net.IP, net.IPv4len)
			newIpAddr    = net.ParseIP("127.0.0.1")
			origPort     uint16
			newPort      uint16 = 6900
			ipAddrOffset        = 47
			portOffset          = 51
		)
		err = m.Get(ipAddrOffset, &origIpAddr)
		if err != nil {
			return nil, err
		}
		log.Printf("origIpAddr: %s", origIpAddr.String())
		log.Printf("replacing with %s", newIpAddr.String())
		m.SetBytes(ipAddrOffset, newIpAddr.To4())
		err = m.Get(portOffset, &origPort)
		if err != nil {
			return nil, err
		}
		log.Printf("origPort: %d", origPort)
		log.Printf("replacing with %d", newPort)
		m.SetUint16(portOffset, newPort)
		remoteAddr = fmt.Sprintf("%s:%d", origIpAddr, origPort)
	}
	if m.ID == 0xac5 {
		var (
			origIpAddr   = make(net.IP, net.IPv4len)
			newIpAddr    = net.ParseIP("127.0.0.1")
			origPort     uint16
			newPort      uint16 = 6900
			ipAddrOffset        = 22
			portOffset          = 26
		)
		err = m.Get(ipAddrOffset, &origIpAddr)
		if err != nil {
			return nil, err
		}
		log.Printf("origIpAddr: %s", origIpAddr.String())
		log.Printf("replacing with %s", newIpAddr.String())
		m.SetBytes(ipAddrOffset, newIpAddr.To4())
		err = m.Get(portOffset, &origPort)
		if err != nil {
			return nil, err
		}
		log.Printf("origPort: %d", origPort)
		log.Printf("replacing with %d", newPort)
		m.SetUint16(portOffset, newPort)
		remoteAddr = fmt.Sprintf("%s:%d", origIpAddr, origPort)
	}

	return m.Body, nil
}

func handleConn(in <-chan *net.TCPConn, out chan<- *net.TCPConn) {
	for conn := range in {
		proxyConn(conn)
		out <- conn
	}
}

func closeConn(in <-chan *net.TCPConn) {
	for conn := range in {
		_ = conn.Close()
	}
}

func main() {
	flag.Parse()

	log.Printf("Listening: %v", localAddr)
	log.Printf("Proxying: %v", remoteAddr)

	localAddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenTCP("tcp", localAddr)
	if err != nil {
		panic(err)
	}

	pending, complete := make(chan *net.TCPConn), make(chan *net.TCPConn)

	for i := 0; i < 5; i++ {
		go handleConn(pending, complete)
	}
	go closeConn(complete)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			panic(err)
		}
		pending <- conn
		log.Println("Queued 1 incoming connection")
	}
}
