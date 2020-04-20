package main

import (
	"fmt"
	"github.com/1xyz/grpc-playground/api"
	"github.com/docopt/docopt-go"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/health"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
	"log"
	"strings"
	"time"
)

/// Don't ever forget to add this line in your import
// _ "google.golang.org/grpc/health"
// terrible, this is the worst

var (
	// see https://github.com/grpc/grpc/blob/master/doc/service_config.md to know more about service config
	retryPolicy = `{
		"methodConfig": [{
		  "name": [{"service": "api.Echo"}],
		  "waitForReady": false,
		  "retryPolicy": {
			  "MaxAttempts": 100,
			  "InitialBackoff": "1s",
			  "MaxBackoff": "100s",
			  "BackoffMultiplier": 2.0,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }
		}]}`

	serviceConfig = `{
		"loadBalancingPolicy": "round_robin",
		"healthCheckConfig": {
			"serviceName": ""
		}
	}`
)

// use grpc.WithDefaultServiceConfig() to set service config
func retryDial(addr string) (*grpc.ClientConn, error) {
	return grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDefaultServiceConfig(retryPolicy))
}

type echoClient struct {
	servers []string

	clientId string
}

func (e *echoClient) OpenSingle() {
	// Set up a connection to the server.
	conn, err := retryDial(e.servers[0])
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()
	c := api.NewEchoClient(conn)
	resp, err := c.FailingEcho(context.Background(), &api.EchoRequest{ClientId: e.clientId})
	if err != nil {
		log.Printf("error = %v", err)
		return
	}

	log.Printf("resp = %v", resp)
}

func (e *echoClient) HealthCheck() {
	// r, cleanup := manual.GenerateAndRegisterManualResolver()
	// defer cleanup()
	//
	// addresses := make([]resolver.Address, 0)
	// for _, s := range e.servers {
	// 	addresses = append(addresses, resolver.Address{Addr: s})
	// }
	//
	// log.Printf("%v\n", addresses)
	//
	// r.InitialState(resolver.State{
	// 	Addresses: addresses,
	// })

	r, cleanup := manual.GenerateAndRegisterManualResolver()
	defer cleanup()
	r.InitialState(resolver.State{
		Addresses: []resolver.Address{
			{Addr: ":8080"},
			{Addr: ":8081"},
		},
	})

	address := fmt.Sprintf("%s:///unused", r.Scheme())

	// address := fmt.Sprintf("%s:///unused", r.Scheme())
	options := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultServiceConfig(serviceConfig),
	}

	conn, err := grpc.Dial(address, options...)
	if err != nil {
		log.Fatalf("did not connect %v", err)
	}
	defer conn.Close()

	echoClient := api.NewEchoClient(conn)
	for {
		callUnaryEcho(echoClient)
		time.Sleep(time.Second)
	}
}

func callUnaryEcho(c api.EchoClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.Echo(ctx, &api.EchoRequest{
		ClientId: uuid.New().String(),
	})

	if err != nil {
		log.Printf("UnaryEcho: _, err = %v\n", err)
	} else {
		log.Printf("UnaryEcho: clock = %v server_id %v \n", r.Clock, r.ServerId)
	}
}

func main() {
	usage := `usage: client [--servers=<servers>]

options:
   --servers=<servers>  Server Addresses [default: :11000,:12000,:13000]..
`

	parser := &docopt.Parser{OptionsFirst: true}
	args, err := parser.ParseArgs(usage, nil, "1.0")
	if err != nil {
		log.Printf("error = %v", err)
		return
	}

	log.Printf("%v\n", args)
	servers, err := args.String("--servers")
	if err != nil {
		log.Printf("err = %v\n", err)
		return
	}

	s := strings.Split(servers, ",")
	for _, e := range s {
		log.Printf("s = %v\n", e)
	}

	cli := &echoClient{
		servers:  s,
		clientId: uuid.New().String(),
	}

	// cli.OpenSingle()
	cli.HealthCheck()
}
