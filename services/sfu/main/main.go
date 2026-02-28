package main

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/sfuserver"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
)

func main() {
	port := 50051
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	sfupb.RegisterSFUServiceServer(grpcServer, sfuserver.NewSFUServer())

	fmt.Printf("SFU gRPC server running on port %d\n", port)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
