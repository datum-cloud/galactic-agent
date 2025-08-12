package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/datum-cloud/galactic-agent/api/local"
)

const DEFAULT_SOCKET_PATH = "/var/run/galactic/agent.sock"

type LocalServer struct {
	local.UnimplementedLocalServer
}

func (s *LocalServer) Register(ctx context.Context, req *local.RegisterRequest) (*local.RegisterReply, error) {
	log.Printf("REGISTER:   vpc='%v', vpcattachment='%v', networks='%v'\n", req.GetVpc(), req.GetVpcattachment(), req.GetNetworks())
	return &local.RegisterReply{Confirmed: true}, nil
}

func (s *LocalServer) Deregister(ctx context.Context, req *local.DeregisterRequest) (*local.DeregisterReply, error) {
	log.Printf("DEREGISTER: vpc='%v', vpcattachment='%v', networks='%v'\n", req.GetVpc(), req.GetVpcattachment(), req.GetNetworks())
	return &local.DeregisterReply{Confirmed: true}, nil
}

func Serve(socketPath string) error {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer listener.Close() //nolint:errcheck

	s := grpc.NewServer()
	local.RegisterLocalServer(s, &LocalServer{})

	reflection.Register(s)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("gRPC listening on unix://%s", socketPath)
		if err := s.Serve(listener); err != nil {
			log.Printf("serve exited: %v", err)
		}
	}()

	<-stop
	log.Println("shutting down...")
	done := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.Stop()
	}
	return nil
}

func main() {
	cmd := &cobra.Command{
		Use:   "galactic-agent",
		Short: "Galactic Agent",
		Run: func(cmd *cobra.Command, args []string) {
			if err := Serve(DEFAULT_SOCKET_PATH); err != nil {
				log.Fatalf("Serve failed: %v", err)
			}
		},
	}
	cmd.SetArgs(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}
