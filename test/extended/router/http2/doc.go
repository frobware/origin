package http2

// This directory has tools to generate:
//
//    .../testdata/router/grpc-interop.yaml.
//
// The generated YAML contains copies of server.go and client.go from
// the cluster directory. The gRPC interop tests require a client and
// server that share a known protobuf implementation.
//
// The files in the cluster directory are zipped up and added as a
// configmap entry in the test setup. The source file are then
// unzipped in an init container and compiled from source. Note: we
// use the go.mod and go.sum files to ensure this is repeatable. Once
// compilation is successful the server is started and listens for
// both h2 and h2c gRPC client connections.

// The client-shell container should be used to invoke gRPC calls
// against either the internal service or via external routes.
