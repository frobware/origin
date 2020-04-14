package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/interop"

	testpb "google.golang.org/grpc/interop/grpc_testing"
)

type testFn func(tc testpb.TestServiceClient, args ...grpc.CallOption)

var (
	listTests = flag.Bool("list-tests", false, "List test case names")
	insecure  = flag.Bool("insecure", false, "Skip TLS")
	caFile    = flag.String("ca-cert", "", "The file containing the CA root cert")
	useTLS    = flag.Bool("tls", false, "Connection uses TLS, if true")
	host      = flag.String("host", "localhost", "host address")
	port      = flag.String("port", "443", "port number")
)

var defaultTestCases = map[string]testFn{
	"cancel_after_begin":          interop.DoCancelAfterBegin,
	"cancel_after_first_response": interop.DoCancelAfterFirstResponse,
	"client_streaming":            interop.DoClientStreaming,
	"custom_metadata":             interop.DoCustomMetadata,
	"empty_stream":                interop.DoEmptyStream,
	"empty_unary":                 interop.DoEmptyUnaryCall,
	"large_unary":                 interop.DoLargeUnaryCall,
	"ping_pong":                   interop.DoPingPong,
	"server_streaming":            interop.DoServerStreaming,
	"special_status_message":      interop.DoSpecialStatusMessage,
	"status_code_and_message":     interop.DoStatusCodeAndMessage,
	"timeout_on_sleeping_server":  interop.DoTimeoutOnSleepingServer,
	"unimplemented_method":        nil, // special case
	"unimplemented_service":       nil, // special case
}

func main() {
	flag.Parse()

	if *listTests {
		for k := range defaultTestCases {
			fmt.Println(k)
		}
		os.Exit(0)
	}

	var opts []grpc.DialOption

	if *useTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: *insecure,
		}
		if *caFile != "" {
			// Avoid auth errors by adding service CA to the rootCA pool. Without leads to:
			//     transport: authentication handshake failed: x509: certificate signed by unknown authority"
			rootCAs, _ := x509.SystemCertPool()
			if rootCAs == nil {
				rootCAs = x509.NewCertPool()
			}

			certs, err := ioutil.ReadFile(*caFile)
			if err != nil {
				grpclog.Fatalf("Failed to append %q to RootCAs: %v", *caFile, err)
			}

			if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
				grpclog.Infoln("No certs appended, using system certs only")
			}

			tlsConfig.RootCAs = rootCAs
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(net.JoinHostPort(*host, *port), append(opts, grpc.WithBlock())...)
	if err != nil {
		grpclog.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	tc := testpb.NewTestServiceClient(conn)
	args := flag.Args()

	if len(args) == 0 {
		for k := range defaultTestCases {
			args = append(args, k)
		}
	}

	for i, name := range args {
		if fn, ok := defaultTestCases[name]; ok && fn != nil {
			fn(tc)
		} else if ok && fn == nil {
			switch name {
			case "unimplemented_method":
				interop.DoUnimplementedMethod(conn)
			case "unimplemented_service":
				interop.DoUnimplementedService(testpb.NewUnimplementedServiceClient(conn))
			}
		} else {
			grpclog.Fatal("Unsupported test case: ", name)
		}
		grpclog.Infof("[#%v/%v] Test %q DONE\n", i+1, len(args), name)
	}
}
