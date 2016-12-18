package mocks

import (
	"fmt"
	"net"
)

type MockProtectedServer struct {
	ServerMessage string
	Port          int
}

// Listen for new clients
func (mps *MockProtectedServer) StartProtectedServer() {
	protectedServerAddr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: mps.Port,
	}

	psListener, _ := net.ListenTCP("tcp", protectedServerAddr)
	go mps.listenForConn(psListener)

}

func (mps *MockProtectedServer) listenForConn(listener *net.TCPListener) {
	for {
		conn, _ := listener.AcceptTCP()
		go mps.handleConn(conn)
	}
}

// Simulate a simple server that reads data and returns something
// This is the server that will be protected and require a proxy to access it
func (mps *MockProtectedServer) handleConn(conn *net.TCPConn) {
	for {
		// Read info from client
		bytes := make([]byte, 4096)
		i, err := conn.Read(bytes)
		if err != nil {
			fmt.Printf("PS: %s\n", err.Error())
			return
		}
		// Write back some fake data
		conn.Write(append([]byte(mps.ServerMessage+": "), bytes[0:i]...))
	}
}
