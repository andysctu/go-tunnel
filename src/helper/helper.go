package helper

import (
	"errors"
	pb "github.com/Originate/go_rps/protobuf"
	"github.com/golang/protobuf/proto"
	"net"
)

func ReceiveProtobuf(conn *net.TCPConn) (*pb.TestMessage, error) {
	if conn == nil {
		return nil, errors.New("Connection closed.")
	}
	bytes := make([]byte, 4096)
	i, err := conn.Read(bytes)
	if err != nil {
		return nil, err
	}
	msg := &pb.TestMessage{}
	if err := proto.Unmarshal(bytes[0:i], msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func GenerateProtobuf(conn *net.TCPConn, userId int32) (*pb.TestMessage, error) {
	if conn == nil {
		return nil, errors.New("Connection closed.")
	}
	// Read info from user
	bytes := make([]byte, 4096)
	i, err := conn.Read(bytes)
	if err != nil {
		return nil, err
	}

	msg := &pb.TestMessage{
		Type: pb.TestMessage_Data,
		Data: bytes[0:i],
		Id:   userId,
	}
	return msg, nil
}
