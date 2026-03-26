package main

import (
	"os"
	"fmt"
	"log"
	"net"
	"strconv"
	
	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"

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
	
        webRTCMediaEngine := &webrtc.MediaEngine{}
        webRTCMediaEngine.RegisterDefaultCodecs()

        interceptorRegistry := &interceptor.Registry{}
        webrtc.RegisterDefaultInterceptors(webRTCMediaEngine, interceptorRegistry)

        webRTCSettingEngine := webrtc.SettingEngine{}
	
	debug := os.Getenv("DEBUG") == "true"
	if !debug {
		public_ip := os.Getenv("PUBLIC_IP")
		webRTCSettingEngine.SetNAT1To1IPs(
				[]string{public_ip},
				webrtc.ICECandidateTypeHost,
		)
	}

	webRTCSettingEngine.SetNetworkTypes([]webrtc.NetworkType{
			webrtc.NetworkTypeUDP4,
	})

	webRTCSettingEngine.SetEphemeralUDPPortRange(50000, 50100)

	webRTCAPI := webrtc.NewAPI(
			webrtc.WithMediaEngine(webRTCMediaEngine),
			webrtc.WithInterceptorRegistry(interceptorRegistry),
			webrtc.WithSettingEngine(webRTCSettingEngine),
	)
	
	maxPeersPerRoom, err := strconv.Atoi(os.Getenv("MAX_PEERS_PER_ROOM"))
	if err != nil {
		log.Fatalf("Failed to get max peers per room: %v", err)
	}
	sfupb.RegisterSFUServiceServer(grpcServer, sfuserver.NewSFUServer(webRTCAPI, maxPeersPerRoom))

	fmt.Println("SFU gRPC server running on ", GRPCAddress)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
