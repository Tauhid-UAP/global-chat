package main

import (
	"os"
	"fmt"
	"log"
	"net"
	
	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	"github.com/Tauhid-UAP/global-chat/services/sfu/internal/sfuserver"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env file not found: %v\n", err)
	}

	GRPCAddress := os.Getenv("GRPC_ADDRESS")
	log.Println("GRPCAddress ", GRPCAddress)
	listener, err := net.Listen("tcp", GRPCAddress)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	sfupb.RegisterSFUServiceServer(grpcServer, sfuserver.NewSFUServer())

	fmt.Println("SFU gRPC server running on ", GRPCAddress)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
