package local

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ImplementedLocalServer struct {
	UnimplementedLocalServer
}

func (s *ImplementedLocalServer) Register(ctx context.Context, req *RegisterRequest) (*RegisterReply, error) {
	log.Printf("REGISTER:   vpc='%v', vpcattachment='%v', networks='%v'\n", req.GetVpc(), req.GetVpcattachment(), req.GetNetworks())
	return &RegisterReply{Confirmed: true}, nil
}

func (s *ImplementedLocalServer) Deregister(ctx context.Context, req *DeregisterRequest) (*DeregisterReply, error) {
	log.Printf("DEREGISTER: vpc='%v', vpcattachment='%v', networks='%v'\n", req.GetVpc(), req.GetVpcattachment(), req.GetNetworks())
	return &DeregisterReply{Confirmed: true}, nil
}

func Serve(socketPath string) error {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer listener.Close() //nolint:errcheck

	s := grpc.NewServer()
	RegisterLocalServer(s, &ImplementedLocalServer{})

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
