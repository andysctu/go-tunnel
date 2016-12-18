package main

import (
	. "github.com/Originate/go_rps/server"
	"log"
	"net"
	"os"
)

// Start rps server
func main() {
	ip := &net.IPAddr{}
	if len(os.Args) < 2 {
		log.Printf("No Host IP provided\n")
		ip.IP = nil
	} else {
		var err error
		ip, err = net.ResolveIPAddr("ip4", os.Args[1])
		if err != nil {
			log.Printf("Invalid Host IP: %s\n", os.Args[1])
			return
		}
	}

	server := GoRpsServer{}
	serverTCPAddr, err := server.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Server running on: %s\n", serverTCPAddr.String())
	select {}
}
