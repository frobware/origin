package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/interop"

	testpb "google.golang.org/grpc/interop/grpc_testing"
)

func main() {
	creds, err := credentials.NewServerTLSFromFile("/etc/serving-certs/tls.crt", "/etc/serving-certs/tls.key")
	if err != nil {
		log.Fatalf("NewServerTLSFromFile failed: %v", err)
	}

	server := grpc.NewServer(grpc.Creds(creds))
	testpb.RegisterTestServiceServer(server, interop.NewTestServer())

	lis, err := net.Listen("tcp", ":8443")
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	log.Printf("Serving h2 at: %v", lis.Addr())

	if err = server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
