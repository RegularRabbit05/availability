package main

import (
	"log"
	"net"
	"time"
)

func initiateConnection(availabilityNode *AvailabilityNode, application *Availability) {
	application.Mutex.Lock()
	availabilityNode.Up = true
	application.Mutex.Unlock()
	for application.Terminate == false {
		if availabilityNode.Up == false {
			time.Sleep(5 * time.Second)
			if application.Terminate == true || application.Reload == true {
				return
			}
			application.Mutex.Lock()
			availabilityNode.Since = time.Now().Unix()
			availabilityNode.Up = true
			application.Mutex.Unlock()
		}

		func() {
			d := net.Dialer{Timeout: 5 * time.Second}
			conn, err := d.Dial("tcp", availabilityNode.IP)
			if err != nil {
				log.Println("Error connecting to node:", err)
				return
			}
			defer conn.Close()
			log.Println("Connected to node:", availabilityNode.IP)

			for application.Terminate == false {
				time.Sleep(1 * time.Second)
				_, err := conn.Write([]byte{0x2E})
				if err != nil {
					return
				}
				buffer := make([]byte, 1)
				_, err = conn.Read(buffer)
				if err != nil {
					return
				}
			}
		}()

		hasUp := false
		application.Mutex.Lock()
		availabilityNode.Up = false
		for i := range application.Nodes {
			if application.Nodes[i].Up == true {
				hasUp = true
				break
			}
		}
		if hasUp == false {
			application.Reload = true
		}
		application.Mutex.Unlock()
	}
}
