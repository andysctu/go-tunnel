package go_rps_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/originate/go_rps/client"
	. "github.com/originate/go_rps/server"
	"github.com/originate/go_rps/test/mocks"
	"io"
	"net"
	"time"
)

var _ = Describe("GoRps", func() {
	var server *GoRpsServer
	var client *GoRpsClient
	var waitTime = 1 * time.Second
	var _ = io.EOF
	var serverTCPAddr *net.TCPAddr
	server1Message := "First server"
	protectedServer := &mocks.MockProtectedServer{
		ServerMessage: server1Message,
		Port:          3000,
	}
	protectedServer.StartProtectedServer()
	var err error
	var exposedPort int
	BeforeEach(func() {
		fmt.Println("----------------")
		server = &GoRpsServer{}
		serverTCPAddr, err = server.Start()
		Expect(err).NotTo(HaveOccurred())

		client = &GoRpsClient{
			ServerTCPAddr: serverTCPAddr,
		}

		err = client.OpenTunnel(protectedServer.Port)
		exposedPort = client.ExposedPort
		Expect(err).NotTo(HaveOccurred())
		Expect(client.ConnToRpsServer).ShouldNot(BeNil())
	})

	AfterEach(func() {
		server = nil
		client = nil
		fmt.Println("----------------")
	})

	Describe("Client stops", func() {
		It("should gracefully stop without error", func() {
			err := client.Stop()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Server stops", func() {
		It("should gracefully stop without error", func() {
			err := server.Stop()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("User tries to connect to a stopped server", func() {
		It("should not allow a connection", func() {
			address := &net.TCPAddr{
				IP:   net.IPv4(127, 0, 0, 1),
				Port: exposedPort,
			}

			// Stop server
			err := server.Stop()
			Expect(err).To(Succeed())

			// Wait for server to stop
			time.Sleep(waitTime)

			// Try to connect to Rps server
			conn, err := net.DialTCP("tcp", nil, address)
			Expect(err).To(HaveOccurred())
			Expect(conn).To(BeNil())
		})
	})

	Describe("User tries to connect to rps server", func() {
		Context("the client is disconnected", func() {
			It("should not allow a connection", func() {
				address := &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: exposedPort,
				}
				err := client.Stop()

				// Wait for client to stop
				time.Sleep(waitTime)

				Expect(err).To(Succeed())

				// Try to connect to Rps server
				conn, err := net.DialTCP("tcp", nil, address)
				Expect(err).To(HaveOccurred())
				Expect(conn).To(BeNil())
			})
		})
	})

	Describe("Client stops after user connects", func() {
		It("should close user's connection", func(done Done) {
			address := &net.TCPAddr{
				IP:   net.IPv4(127, 0, 0, 1),
				Port: exposedPort,
			}

			// Connect to Rps server
			conn, err := net.DialTCP("tcp", nil, address)
			Expect(err).NotTo(HaveOccurred())

			// Wait for user to connect
			time.Sleep(waitTime)

			err = client.Stop()
			Expect(err).NotTo(HaveOccurred())

			// Wait for client to stop
			time.Sleep(waitTime)

			// Try to read the response
			bytes := make([]byte, 4096)
			i, err := conn.Read(bytes)
			Expect(i).To(Equal(0))
			Expect(err).To(Equal(io.EOF))
			close(done)
		}, 5)
	})

	Describe("Server stops after user connects", func() {
		It("should gracefully stop without error", func(done Done) {
			address := &net.TCPAddr{
				IP:   net.IPv4(127, 0, 0, 1),
				Port: exposedPort,
			}
			// Connect to Rps server
			conn, err := net.DialTCP("tcp", nil, address)
			Expect(err).NotTo(HaveOccurred())

			// Let the connection establish
			time.Sleep(waitTime)

			err = server.Stop()
			Expect(err).NotTo(HaveOccurred())

			// Try to read the response
			bytes := make([]byte, 4096)
			i, err := conn.Read(bytes)
			Expect(i).To(Equal(0))
			Expect(err).To(Equal(io.EOF))
			close(done)
		}, 5)
	})

	Describe("1 user connects to rps server", func() {
		Context("user sends a message and disconnects", func() {
			It("should gracefully stop without error", func() {
				address := &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: exposedPort,
				}

				// Connect to Rps server
				userConn, err := net.DialTCP("tcp", nil, address)
				Expect(err).NotTo(HaveOccurred())

				_, err = userConn.Write([]byte("Hello world"))
				Expect(err).NotTo(HaveOccurred())

				// Read the response
				bytes := make([]byte, 4096)
				i, err := userConn.Read(bytes)
				Expect(err).NotTo(HaveOccurred())

				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server1Message + ": Hello world")))

				err = userConn.Close()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("A user hitting the rps server", func() {
		Context("to access the protected server", func() {
			It("should forward user data to the client, then the protected server", func(done Done) {
				// _, exposedPort := client.OpenTunnel(3000)
				address := &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: exposedPort,
				}

				// Connect to Rps server
				userConn, err := net.DialTCP("tcp", nil, address)
				Expect(err).NotTo(HaveOccurred())

				// Send some data
				userConn.Write([]byte("Hello world"))

				// Read the response
				bytes := make([]byte, 4096)
				i, err := userConn.Read(bytes)
				Expect(err).NotTo(HaveOccurred())

				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server1Message + ": Hello world")))
				userConn.Close()
				close(done)
			}, 10)
		})
	})

	Describe("A user hitting the rps server", func() {
		Context("sending two messages", func() {
			It("should successfully get both messages to the protected server", func(done Done) {
				// _, exposedPort := client.OpenTunnel(3000)
				address := &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: exposedPort,
				}

				// Connect to Rps server
				userConn, err := net.DialTCP("tcp", nil, address)
				Expect(err).NotTo(HaveOccurred())

				// Send msg 1
				userConn.Write([]byte("Message 1"))

				// Read the response
				bytes := make([]byte, 4096)
				i, err := userConn.Read(bytes)
				Expect(err).NotTo(HaveOccurred())
				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server1Message + ": Message 1")))

				// Send msg 2
				userConn.Write([]byte("Message 2"))

				// Read the response
				bytes = make([]byte, 4096)
				i, err = userConn.Read(bytes)
				Expect(err).NotTo(HaveOccurred())
				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server1Message + ": Message 2")))
				userConn.Close()
				close(done)
			}, 10)
		})
	})

	Describe("Two users hitting the rps server", func() {
		Context("to access the same protected server", func() {
			It("should forward users' datum to the client, then the protected server", func(done Done) {
				// _, exposedPort := client.OpenTunnel(3000)
				address := &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: exposedPort,
				}

				// First user connects to Rps server
				userConn0, err := net.DialTCP("tcp", nil, address)
				Expect(err).NotTo(HaveOccurred())

				// Second user connects to Rps server
				userConn1, err := net.DialTCP("tcp", nil, address)
				Expect(err).NotTo(HaveOccurred())

				// First user sends some data
				userConn0.Write([]byte("Hello from user0"))

				// First user reads the response
				bytes := make([]byte, 4096)
				i, err := userConn0.Read(bytes)
				Expect(err).NotTo(HaveOccurred())
				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server1Message + ": Hello from user0")))
				err = userConn0.Close()
				Expect(err).To(Succeed())

				// Second user sends some data
				userConn1.Write([]byte("Hello from user1"))

				// Second user reads the response
				bytes = make([]byte, 4096)
				i, err = userConn1.Read(bytes)
				Expect(err).NotTo(HaveOccurred())
				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server1Message + ": Hello from user1")))

				err = userConn1.Close()
				Expect(err).To(Succeed())
				close(done)
			}, 5)
		})
	})

	Describe("Two users connect to rps server", func() {
		Context("to access two different protected servers", func() {
			It("should successfully deliver data", func(done Done) {
				server2Message := "Second server"
				protectedServer2 := &mocks.MockProtectedServer{
					ServerMessage: server2Message,
					Port:          3001,
				}
				protectedServer2.StartProtectedServer()

				address1 := &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: exposedPort,
				}

				client2 := &GoRpsClient{
					ServerTCPAddr: serverTCPAddr,
				}

				err = client2.OpenTunnel(protectedServer2.Port)
				exposedPort2 := client2.ExposedPort
				Expect(err).NotTo(HaveOccurred())
				Expect(client.ConnToRpsServer).ShouldNot(BeNil())

				address2 := &net.TCPAddr{
					IP:   net.IPv4(127, 0, 0, 1),
					Port: exposedPort2,
				}

				// First user connects to Rps server
				userConn0, err := net.DialTCP("tcp", nil, address1)
				Expect(err).NotTo(HaveOccurred())

				// Second user connects to Rps server
				userConn1, err := net.DialTCP("tcp", nil, address2)
				Expect(err).NotTo(HaveOccurred())

				// First user sends some data
				userConn0.Write([]byte("Hello from user0"))

				// First user reads the response
				bytes := make([]byte, 4096)
				i, err := userConn0.Read(bytes)
				Expect(err).NotTo(HaveOccurred())
				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server1Message + ": Hello from user0")))
				err = userConn0.Close()
				Expect(err).To(Succeed())

				// Second user sends some data
				userConn1.Write([]byte("Hello from user1"))

				// Second user reads the response
				bytes = make([]byte, 4096)
				i, err = userConn1.Read(bytes)
				Expect(err).NotTo(HaveOccurred())
				// Should be the response from the simulated protected server
				Expect(bytes[0:i]).To(Equal([]byte(server2Message + ": Hello from user1")))

				err = userConn1.Close()
				Expect(err).To(Succeed())
				close(done)
			}, 5)
		})
	})
})
