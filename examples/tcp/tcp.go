package tcp

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/tun/netstack"
)

func TestSendAndReceiveOverTCP(localIP string, localPort int, remoteIP string, remotePort int, vNet *netstack.Net) error {
	if err := sendMessage(vNet, remoteIP, remotePort, "hello-world"); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	message, err := receiveMessage(vNet, localIP, localPort)
	if err != nil {
		return fmt.Errorf("failed to receive message: %v", err)
	}

	log.Printf("Received message: %s", message)
	return nil
}

func TestReceiveAndSendOverTCP(localIP string, localPort int, remoteIP string, remotePort int, vNet *netstack.Net) error {
	message, err := receiveMessage(vNet, localIP, localPort)
	if err != nil {
		return fmt.Errorf("failed to receive message: %v", err)
	}

	log.Printf("Received message: %s", message)

	if err := sendMessage(vNet, remoteIP, remotePort, "hello-back"); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

func sendMessage(vNet *netstack.Net, remoteIP string, remotePort int, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := vNet.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", remoteIP, remotePort))
	if err != nil {
		return fmt.Errorf("failed to connect to other node: %v", err)
	}
	defer conn.Close()

	log.Printf("Sending message: %s", message)
	_, err = conn.Write([]byte(message))
	return err
}

func receiveMessage(vNet *netstack.Net, localIP string, localPort int) (string, error) {
	listener, err := vNet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(localIP), Port: localPort})
	if err != nil {
		return "", fmt.Errorf("failed to listen on TCP: %v", err)
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		return "", fmt.Errorf("failed to accept connection: %v", err)
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read from connection: %v", err)
	}

	return string(buffer[:n]), nil
}
