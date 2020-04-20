package main

import (
	"github.com/1xyz/grpc-playground/api"
	"github.com/docopt/docopt-go"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"time"
)

type EchoServer struct {
	api.UnimplementedEchoServer

	id string

	tickDuration time.Duration

	shutdownCh chan bool

	isLeader bool
}

func (es *EchoServer) Echo(ctx context.Context, req *api.EchoRequest) (*api.EchoResponse, error) {
	log.Printf("echo req = %v\n", req)
	return &api.EchoResponse{
		ServerId: es.id,
		ClientId: req.ClientId,
		Clock:    now(),
	}, nil
}

func (es *EchoServer) StreamEcho(req *api.EchoRequest, stream api.Echo_StreamEchoServer) error {
	ticker := time.NewTicker(es.tickDuration)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			sec := t.UTC().Unix()
			stream.Send(&api.EchoResponse{
				ServerId: es.id,
				ClientId: req.ClientId,
				Clock:    sec,
			})

		case <-es.shutdownCh:
			return nil
		}
	}
}

func (es *EchoServer) FailingEcho(ctx context.Context, req *api.EchoRequest) (*api.EchoResponse, error) {
	log.Printf("echo req = %v\n", req)
	return nil, status.Error(codes.Unavailable, "I am just gonna fail")
}

func (es *EchoServer) IsLeader(ctx context.Context, e *api.Empty) (*api.IsLeaderResponse, error) {
	return &api.IsLeaderResponse{IsLeader: es.isLeader}, nil
}

func now() int64 {
	return time.Now().UTC().Unix()
}

func main() {
	usage := `usage: server [--address=<address>] [--leader]

options:
   --address=<address>  Listen Address [default: :11000]..
   --leader             Is leader.
`
	parser := &docopt.Parser{OptionsFirst: true}
	args, err := parser.ParseArgs(usage, nil, "1.0")
	if err != nil {
		log.Printf("error = %v", err)
		return
	}

	log.Printf("%v\n", args)
	addr, err := args.String("--address")
	if err != nil {
		log.Printf("err = %v\n", err)
		return
	}

	isLeader, err := args.Bool("--leader")
	if err != nil {
		log.Printf("err = %v\n", err)
		return
	}

	grpcServer := grpc.NewServer()
	api.RegisterEchoServer(grpcServer, newEchoServer(isLeader))

	log.Printf("addr = %v\n", addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("started...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Panic(err)
	}
}

func newEchoServer(isLeader bool) *EchoServer {
	a := &EchoServer{
		id:           uuid.New().String(),
		tickDuration: time.Second,
		shutdownCh:   make(chan bool),
		isLeader:     isLeader,
	}

	log.Printf("new server id = %v isLeader = %v\n",
		a.isLeader, a.isLeader)

	return a
}
