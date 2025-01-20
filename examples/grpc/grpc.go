package grpc

import (
	context "context"
	"fmt"
	"log"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/tun/netstack"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type srv struct {
	UnimplementedTestGRPCServer
}

func (s *srv) Test(ctx context.Context, req *TestRequest) (*TestResponse, error) {
	log.Printf("Server received message: %s", req.Msg)
	return &TestResponse{Msg: "hello-back"}, nil
}

func TestServer(ip string, port int, vNet *netstack.Net) error {
	lis, err := vNet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(ip), Port: port})
	if err != nil {
		return fmt.Errorf("unable to listen on %s:%d: %w", ip, port, err)
	}
	defer lis.Close()

	srv := &srv{}

	grpcServer := grpc.NewServer()
	RegisterTestGRPCServer(grpcServer, srv)

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve gRPC server: %v", err)
		}
	}()

	log.Println("grpc server is running")
	select {}
}

func TestClient(ip string, port int, vNet *netstack.Net) error {
	commcheckAddress := fmt.Sprintf("%s:%d", ip, port)

	conn, err := grpc.NewClient(commcheckAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return vNet.DialContext(ctx, "tcp", addr)
		}),
	)
	if err != nil {
		return fmt.Errorf("grpc client failed to dial grpc server: %w", err)
	}
	defer conn.Close()

	client := NewTestGRPCClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	r, err := client.Test(ctx, &TestRequest{Msg: "hello"})
	if err != nil {
		return fmt.Errorf("grpc client failed to send message to grpc server: %w", err)
	}

	log.Printf("Received message: %s", r.Msg)

	return nil
}
