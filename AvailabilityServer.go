package main

import (
	"net"
	"time"
)

func handleConnection(conn net.Conn, application *Availability) {
	defer conn.Close()
	const deadline = 5
	for application.Terminate == false {
		conn.SetReadDeadline(time.Now().Add(deadline * time.Second))
		buffer := make([]byte, 1)
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}
		time.Sleep(1 * time.Second)
		conn.SetWriteDeadline(time.Now().Add(deadline * time.Second))
		_, err = conn.Write([]byte{0x2E})
		if err != nil {
			return
		}
	}
}

func handleListener(listener net.Listener, application *Availability) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go handleConnection(conn, application)
	}
}
