package main

import (
	"github.com/1xyz/grpc-playground/api"
	"github.com/docopt/docopt-go"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"strings"
)

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

	cli.OpenSingle()
}
