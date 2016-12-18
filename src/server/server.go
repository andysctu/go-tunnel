package server

import (
	"fmt"
	"github.com/Originate/go_rps/helper"
	pb "github.com/Originate/go_rps/protobuf"
	"github.com/golang/protobuf/proto"
	"io"
	"log"
	"net"
	"strconv"
)

type GoRpsServer struct {
	UserConn map[int32]*net.TCPConn
	UserId   map[*net.TCPConn]int32

	clientToUserConn     map[*net.TCPConn][]*net.TCPConn
	clientToUserListener map[*net.TCPConn]*net.TCPListener
	clientListener       *net.TCPListener
}

func (s *GoRpsServer) Start() (*net.TCPAddr, error) {
	s.UserConn = make(map[int32]*net.TCPConn)
	s.UserId = make(map[*net.TCPConn]int32)
	s.clientToUserConn = make(map[*net.TCPConn][]*net.TCPConn)
	s.clientToUserListener = make(map[*net.TCPConn]*net.TCPListener)

	address := &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 34567,
	}

	var err error
	s.clientListener, err = net.ListenTCP("tcp", address)
	if err != nil {
		return nil, err
	}

	// Listen for clients
	go s.listenForClients()

	// Convert net.Addr to *net.TCPAddr and return
	clientListenerAddr, err := net.ResolveTCPAddr("tcp", s.clientListener.Addr().String())
	if err != nil {
		return nil, err
	}
	return clientListenerAddr, nil
}

func (s *GoRpsServer) listenForClients() {
	log.Printf("RPS Server listening for clients on: %s\n", s.clientListener.Addr().String())
	for {
		// Blocks until a client connects
		clientConn, err := s.clientListener.AcceptTCP()
		if err != nil {
			return
		}
		go s.handleClientConn(clientConn)

		// Choose a random free port to expose to users
		address := &net.TCPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		}

		// Create a listener for that port, and extract the chosen port
		userListener, err := net.ListenTCP("tcp", address)
		addr, err := net.ResolveTCPAddr("tcp", userListener.Addr().String())
		exposedPort := addr.Port

		// Tell client the exposed port
		portStr := strconv.Itoa(exposedPort)
		msg := &pb.TestMessage{
			Type: pb.TestMessage_ConnectionOpen,
			Data: []byte(portStr),
			Id:   -1,
		}
		bytes, err := proto.Marshal(msg)
		if err != nil {
			log.Println(err.Error())
			return
		}

		// Tell the client what port is exposed to users for their connection
		clientConn.Write(bytes)

		// Each client is associated with one user listener, and possibly multiple users
		s.clientToUserListener[clientConn] = userListener

		// Start listening for users on that port, for the new client
		go s.listenForUsers(userListener, exposedPort, clientConn)
	}
}

func (s *GoRpsServer) listenForUsers(userListener *net.TCPListener, exposedPort int, clientConn *net.TCPConn) {
	log.Printf("Server listening for users on: %s\n", userListener.Addr().String())
	for {
		// Listen for a user connection
		userConn, err := userListener.AcceptTCP()
		if err != nil {
			log.Println(err.Error())
			return
		}

		log.Println("User connection established")

		// User memory address as ID
		addrStr := fmt.Sprintf("%x", userConn)
		id, err := strconv.ParseInt(addrStr[3:len(addrStr)-2], 16, 64)
		if err != nil {
			log.Printf("Error converting addr to id: %s\n", err.Error())
			continue
		}
		id32 := int32(id)

		s.UserConn[id32] = userConn
		s.UserId[userConn] = id32
		s.clientToUserConn[clientConn] = append(s.clientToUserConn[clientConn], userConn)

		// Tell client to open a connection for user <id>
		msg := &pb.TestMessage{
			Type: pb.TestMessage_ConnectionOpen,
			Id:   id32,
			Data: []byte(pb.TestMessage_ConnectionOpen.String()),
		}
		sendToClient(msg, clientConn)

		go s.handleUserConn(userConn, clientConn)
	}
}

func (s *GoRpsServer) Stop() (err error) {
	// Close all user listeners first
	for _, userListener := range s.clientToUserListener {
		err = userListener.Close()
		if err != nil {
			return err
		}
	}

	// Close the client listener
	err = s.clientListener.Close()
	if err != nil {
		log.Printf("Error closing client listener: %s\n", err.Error())
	}

	// Close all existing client connections and their associated user connections
	for clientConn, userConns := range s.clientToUserConn {
		for _, userConn := range userConns {
			err = userConn.Close()
			if err != nil {
				log.Printf("Error closing user conn: %s\n", err.Error())
				return err
			}
		}
		err = clientConn.Close()
		if err != nil {
			log.Printf("Error closing client conn: %s\n", err.Error())
			return err
		}
	}

	s.clientToUserConn = make(map[*net.TCPConn][]*net.TCPConn)
	s.clientToUserListener = make(map[*net.TCPConn]*net.TCPListener)
	return nil
}

func (s *GoRpsServer) handleClientConn(clientConn *net.TCPConn) {
	for {
		// Blocks until we receive some data from client
		msg, err := helper.ReceiveProtobuf(clientConn)
		if err != nil {
			if err == io.EOF {
				err = clientConn.Close()
				if err != nil {
					log.Printf("Error closing client connection: %s\n", clientConn)
				}
				s.clientDisconnected(clientConn)
				return
			}
			log.Printf("Error receiving from client: %s\n", err.Error())
			return
		}

		switch msg.Type {
		// Client told us that protected server has disconnected
		// We need to disconnect all users associated with this client
		case pb.TestMessage_ConnectionClose:
			{
				// Close user listener
				err = s.clientToUserListener[clientConn].Close()
				if err != nil {
					log.Printf("Error closing user listener: %s\n", err.Error())
				}
				delete(s.clientToUserListener, clientConn)

				// Close all user connections associated with client
				for _, userConn := range s.clientToUserConn[clientConn] {
					err = userConn.Close()
					if err != nil {
						log.Printf("Error closing connection for user <%d>: %s\n", s.UserId[userConn], err.Error())
					}
				}
				delete(s.clientToUserConn, clientConn)

				// Close client connection
				err = clientConn.Close()
				if err != nil {
					log.Printf("Error closing connection for client: %s\n", err.Error())
				}
				return
			}

		// Forward data from client to user
		case pb.TestMessage_Data:
			{
				_, err = s.UserConn[msg.Id].Write([]byte(msg.Data))
				if err != nil {
					log.Printf("Error writing to user: %s\n", err.Error())
					return
				}
				break
			}
		}

	}
}

func (s *GoRpsServer) handleUserConn(userConn *net.TCPConn, clientConn *net.TCPConn) {
	userId := s.UserId[userConn]
	for {
		// Blocks until we receive data from user
		// Generates a protobuf msg with the user's data as the msg.Data field
		msg, err := helper.GenerateProtobuf(userConn, userId)
		if err != nil {
			if err == io.EOF {
				log.Printf("User <%d> has disconnected.\n", userId)
				err = userConn.Close()
				if err != nil {
					log.Printf("Error closing connection for user <%d>: %s\n", userId, err.Error())
					return
				}
				log.Printf("User <%d> connection successfully closed.\n", userId)
				s.userDisconnected(userId, clientConn)
				return
			}
			log.Printf("Error receving from user: %s\n", err.Error())
			return
		}

		// Forward data to associated client
		err = sendToClient(msg, clientConn)
		if err != nil {
			log.Printf("Error forwarding data to client: %s\n", err.Error())
		}
	}
}

func (s *GoRpsServer) userDisconnected(userId int32, clientConn *net.TCPConn) {
	// Tell client a user disconnected
	msg := &pb.TestMessage{
		Type: pb.TestMessage_ConnectionClose,
		Data: []byte(pb.TestMessage_ConnectionClose.String()),
		Id:   userId,
	}
	err := sendToClient(msg, clientConn)
	if err != nil {
		log.Printf("Error forwarding data to client: %s\n", err.Error())
	}
}

func (s *GoRpsServer) clientDisconnected(clientConn *net.TCPConn) {
	// Close user listener associated with client
	err := s.clientToUserListener[clientConn].Close()
	if err != nil {
		log.Printf("Error closing user listener: %s\n", err.Error())
	}

	// Disconnect all users associated with client
	for _, userConn := range s.clientToUserConn[clientConn] {
		err := userConn.Close()
		if err != nil {
			log.Printf("Error closing connection for user <%d>\n", s.UserId[userConn])
		}
	}
}

func sendToClient(msg *pb.TestMessage, clientConn *net.TCPConn) error {
	out, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	// Forward data to the associated client
	_, err = clientConn.Write(out)
	return err
}
