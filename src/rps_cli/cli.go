package main

import (
	. "github.com/Originate/go_rps/client"
	"github.com/codegangsta/cli"
	"log"
	"net"
	"os"
	"strconv"
)

func main() {
	app := cli.NewApp()
	app.Name = "rps_cli"
	app.Usage = "Expose a local server hidden behind a firewall"
	app.Action = func(c *cli.Context) error {
		portStr := c.Args()[0]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Printf("Invalid port: %s\n", portStr)
			return nil
		}
		log.Printf("Exposing whatever is currently running on port: %d\n", port)

		serverTCPAddrStr := c.Args()[1]
		serverTCPAddr, err := net.ResolveTCPAddr("tcp", serverTCPAddrStr)
		log.Printf("Connecting to rps server @: %s\n", serverTCPAddrStr)
		if err != nil {
			log.Printf("Invalid server address: %s\n", serverTCPAddrStr)
			return nil
		}

		client := GoRpsClient{
			ServerTCPAddr: serverTCPAddr,
		}

		err = client.OpenTunnel(port)
		if err != nil {
			log.Printf("Unable to open tunnel.\n")
			return nil
		}

		exposedTCPAddr := *serverTCPAddr
		exposedTCPAddr.Port = client.ExposedPort
		log.Printf("Tunnel opened! Go here: %s\n", exposedTCPAddr.String())
		select {}
	}
	app.Run(os.Args)
}
