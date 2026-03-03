package sfuclient

import (
	"context"
	"sync"
	"log"

	sfupb "github.com/Tauhid-UAP/global-chat/proto/sfu"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SFUClient struct {
	Conn *grpc.ClientConn
	ServiceClient sfupb.SFUServiceClient
}

func NewSFUClient(sfuAddress string) (*SFUClient, error) {
	conn, err := grpc.Dial(
		sfuAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	serviceClient := sfupb.NewSFUServiceClient(conn)

	return &SFUClient{
		Conn: conn,
		ServiceClient: serviceClient,
	}, nil
}

func (sfucl *SFUClient) Close() {
	conn := sfucl.Conn
	if conn == nil {
		return
	}

	conn.Close()
}

type UserSignalStream struct {
	Stream sfupb.SFUService_SignalClient
	Ctx context.Context
	Cancel context.CancelFunc
	mu sync.Mutex
}

func (sfucl *SFUClient) CreateUserStream() (*UserSignalStream, error) {
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := sfucl.ServiceClient.Signal(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	return &UserSignalStream{
		Stream: stream,
		Ctx: ctx,
		Cancel: cancel,
	}, nil
}

func (ust *UserSignalStream) Send(req *sfupb.SignalRequest) error {
	ust.mu.Lock()
	defer ust.mu.Unlock()
	return ust.Stream.Send(req)
}

func (ust *UserSignalStream) Close() {
	ust.Cancel()
	log.Println("Closed gRPC stream")
}
