package main

import (
	"github.com/1xyz/grpc-playground/api"
	"github.com/docopt/docopt-go"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
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

// ////////////////////////////////////////////////////////////////////////////////////////

type HealthCheckServer struct {
	healthgrpc.UnimplementedHealthServer

	echoServer *EchoServer

	tickDuration time.Duration

	shutdownCh chan bool
}

func (h *HealthCheckServer) Check(ctx context.Context,
	req *healthgrpc.HealthCheckRequest) (*healthgrpc.HealthCheckResponse, error) {
	log.Printf("check: req svc: %v", req.Service)

	s := healthgrpc.HealthCheckResponse_UNKNOWN
	if h.echoServer == nil {
		s = healthgrpc.HealthCheckResponse_UNKNOWN
	} else {
		if h.echoServer.isLeader {
			s = healthgrpc.HealthCheckResponse_SERVING
		} else {
			s = healthgrpc.HealthCheckResponse_NOT_SERVING
		}
	}

	return &healthgrpc.HealthCheckResponse{
		Status: s,
	}, nil
}

func (h *HealthCheckServer) Watch(req *healthgrpc.HealthCheckRequest, stream healthgrpc.Health_WatchServer) error {
	ticker := time.NewTicker(h.tickDuration)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			resp, err := h.Check(stream.Context(), req)
			if err != nil {
				return err
			}

			log.Printf("sending resp at %v resp=%v", t, resp)
			stream.Send(resp)

		case <-h.shutdownCh:
			return nil
		}
	}
}

// ////////////////////////////////////////////////////////////////////////////////////////

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

	s := grpc.NewServer()
	echoServer := newEchoServer(isLeader)
	api.RegisterEchoServer(s, echoServer)
	healthcheck := newHealthCheckServer(echoServer)
	healthgrpc.RegisterHealthServer(s, healthcheck)

	log.Printf("addr = %v\n", addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("started...")
	if err := s.Serve(lis); err != nil {
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

func newHealthCheckServer(server *EchoServer) *HealthCheckServer {
	h := &HealthCheckServer{
		echoServer:   server,
		tickDuration: time.Second,
		shutdownCh:   make(chan bool),
	}

	log.Printf("created health checker server\n")
	return h
}
