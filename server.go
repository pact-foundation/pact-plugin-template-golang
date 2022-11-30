package main

// This file contains the base gRPC server implementation
// You probably don't need to change this

import (
	"fmt"
	"log"
	"net"

	plugin "github.com/pact-foundation/pact-plugin-template-golang/io_pact_plugin"
	"google.golang.org/grpc"
)

type serverDetails struct {
	Port      int
	ServerKey string
}

func startPluginServer(details serverDetails) {
	log.Println("starting server on port", details.Port)

	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", details.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Required JSON structure for plugin framework to
	fmt.Printf(`{"port": %d, "serverKey": "%s"}%s`, details.Port, details.ServerKey, "\n")

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	plugin.RegisterPactPluginServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}

func newServer() *pluginServer {
	s := &pluginServer{}
	return s
}

type pluginServer struct {
	plugin.UnimplementedPactPluginServer
}
