package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type AvailabilityNode struct {
	IP string
	Up bool
}

type Availability struct {
	Nodes     []AvailabilityNode
	Reload    bool
	Mutex     sync.Mutex
	Terminate bool
}

func fetchNodeConfig(configFile string) ([]string, []string, error) {
	type NodeConfig struct {
		Nodes    []string `json:"nodes"`
		Commands []string `json:"commands"`
	}
	var conf NodeConfig

	resp, err := http.Get(configFile)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&conf)
	if err != nil {
		return nil, nil, err
	}

	return conf.Nodes, conf.Commands, nil
}

func startInterface(ip string, port int, application *Availability) {
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		application.Mutex.Lock()
		defer application.Mutex.Unlock()
		type NodeStatus struct {
			IP string `json:"ip"`
			Up bool   `json:"up"`
		}
		var status []NodeStatus
		for _, node := range application.Nodes {
			status = append(status, NodeStatus{IP: node.IP, Up: node.Up})
		}
		json.NewEncoder(w).Encode(status)
	})
	log.Printf("Starting interface on port %d\n", port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", ip, port), nil)
	if err != nil {
		log.Fatal("Interface error:", err)
	}
}

func main() {
	if len(os.Args)-1 < 3 {
		fmt.Printf("%s <nodes-url> <listen-ip> <common-port> [interface-port]", os.Args[0])
		fmt.Println()
		os.Exit(1)
	}

	var application = new(Availability)
	application.Reload = false
	serverPort, err := strconv.Atoi(os.Args[3])
	if serverPort < 1 || serverPort > 65535 || err != nil {
		log.Fatal("Invalid common port number\n")
	}

	if len(os.Args)-1 >= 4 {
		interfacePort, err := strconv.Atoi(os.Args[4])
		if interfacePort < 1 || interfacePort > 65535 || err != nil {
			log.Fatal("Invalid interface port number\n")
		}
		go startInterface(os.Args[2], interfacePort, application)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", os.Args[2], serverPort))
	if err != nil {
		log.Fatal("Error listening:", err, "\n")
	}
	defer listener.Close()
	go handleListener(listener, application)

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownSignal
		log.Println("CTRL+C detected. Shutting down...")
		application.Mutex.Lock()
		application.Terminate = true
		application.Mutex.Unlock()
		time.Sleep(10 * time.Second)
		os.Exit(-1)
	}()

	log.Println("Loading nodes...")
	for !application.Terminate {
		nodes, commands, err := fetchNodeConfig(os.Args[1])
		if err != nil {
			log.Println("Failed to download Nodes")
			for i := 0; i < 10; i++ {
				if application.Terminate == true {
					break
				}
				time.Sleep(1 * time.Second)
			}
			continue
		}

		application.Mutex.Lock()
		application.Nodes = make([]AvailabilityNode, 0)
		for _, nodeIP := range nodes {
			if strings.ToLower(nodeIP) == strings.ToLower(os.Args[2]) {
				continue
			}
			application.Nodes = append(application.Nodes, AvailabilityNode{IP: nodeIP + ":" + strconv.Itoa(serverPort), Up: true})
		}
		application.Mutex.Unlock()

		log.Printf("Loaded %d nodes\n", len(application.Nodes))

		var wg sync.WaitGroup
		for i := range application.Nodes {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				initiateConnection(&application.Nodes[i], application)
			}(i)
		}
		wg.Wait()

		application.Reload = false
		application.Mutex.Lock()
		isTerminating := application.Terminate
		application.Mutex.Unlock()

		if !isTerminating {
			log.Println("All nodes have disconnected at the same time. Assuming connection has been lost...")
			for i := 0; i < 5; i++ {
				if application.Terminate == true {
					break
				}
				time.Sleep(1 * time.Second)
			}
			for _, command := range commands {
				log.Println("Executing command:", command)
				parts := strings.Fields(command)
				head := parts[0]
				parts = parts[1:]
				cmd := exec.Command(head, parts...)
				cmd.Run()
				time.Sleep(1 * time.Second)
			}
			for i := 0; i < 5; i++ {
				if application.Terminate == true {
					break
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
}
