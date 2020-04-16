package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/interop"

	testpb "google.golang.org/grpc/interop/grpc_testing"
)

var (
	insecure = flag.Bool("insecure", false, "Skip certificate verification")
	caFile   = flag.String("ca-cert", "", "The file containing the CA root cert")
	useTLS   = flag.Bool("tls", false, "Connection uses TLS, if true")
	host     = flag.String("host", "localhost", "host address")
	port     = flag.Int("port", 8443, "port number")
)

type DialParams struct {
	UseTLS   bool
	CertData []byte
	Host     string
	Port     int
	Insecure bool
}

func Dial(cfg DialParams) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if cfg.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.Insecure,
		}
		if len(cfg.CertData) > 0 {
			rootCAs, err := x509.SystemCertPool()
			if err != nil {
				return nil, err
			}
			if rootCAs == nil {
				rootCAs = x509.NewCertPool()
			}
			if ok := rootCAs.AppendCertsFromPEM(cfg.CertData); !ok {
				return nil, errors.New("failed to append certs")
			}
			tlsConfig.RootCAs = rootCAs
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	return grpc.Dial(net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), append(opts, grpc.WithBlock())...)
}

func main() {
	flag.Parse()

	dialParams := DialParams{
		UseTLS:   *useTLS,
		Host:     *host,
		Port:     *port,
		Insecure: *insecure,
	}

	if *caFile != "" {
		certs, err := ioutil.ReadFile(*caFile)
		if err != nil {
			log.Fatalf("Failed to read %q: %v", *caFile, err)
		}
		dialParams.CertData = certs
	}

	log.Printf("Dial params: %+v", dialParams)

	for _, testCase := range flag.Args() {
		conn, err := Dial(dialParams)
		if err != nil {
			log.Fatalf("Dial failed: %v", err)
		}

		tc := testpb.NewTestServiceClient(conn)

		switch testCase {
		case "empty_unary":
			interop.DoEmptyUnaryCall(tc)
			grpclog.Infoln("EmptyUnaryCall done")
		case "large_unary":
			interop.DoLargeUnaryCall(tc)
			grpclog.Infoln("LargeUnaryCall done")
		case "client_streaming":
			interop.DoClientStreaming(tc)
			grpclog.Infoln("ClientStreaming done")
		case "server_streaming":
			interop.DoServerStreaming(tc)
			grpclog.Infoln("ServerStreaming done")
		case "ping_pong":
			interop.DoPingPong(tc)
			grpclog.Infoln("Pingpong done")
		case "empty_stream":
			interop.DoEmptyStream(tc)
			grpclog.Infoln("Emptystream done")
		case "timeout_on_sleeping_server":
			interop.DoTimeoutOnSleepingServer(tc)
			grpclog.Infoln("TimeoutOnSleepingServer done")
		case "cancel_after_begin":
			interop.DoCancelAfterBegin(tc)
			grpclog.Infoln("CancelAfterBegin done")
		case "cancel_after_first_response":
			interop.DoCancelAfterFirstResponse(tc)
			grpclog.Infoln("CancelAfterFirstResponse done")
		case "status_code_and_message":
			interop.DoStatusCodeAndMessage(tc)
			grpclog.Infoln("StatusCodeAndMessage done")
		case "special_status_message":
			interop.DoSpecialStatusMessage(tc)
			grpclog.Infoln("SpecialStatusMessage done")
		case "custom_metadata":
			interop.DoCustomMetadata(tc)
			grpclog.Infoln("CustomMetadata done")
		case "unimplemented_method":
			interop.DoUnimplementedMethod(conn)
			grpclog.Infoln("UnimplementedMethod done")
		case "unimplemented_service":
			interop.DoUnimplementedService(testpb.NewUnimplementedServiceClient(conn))
			grpclog.Infoln("UnimplementedService done")
		case "pick_first_unary":
			interop.DoPickFirstUnary(tc)
			grpclog.Infoln("PickFirstUnary done")
		default:
			grpclog.Fatal("Unsupported test case: ", testCase)
		}
	}
}
