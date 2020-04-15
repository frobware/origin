package grpc_interop

import (
	"fmt"

	"google.golang.org/grpc"
	testpb "google.golang.org/grpc/interop/grpc_testing"
)

var defaultTestCases = []string{
	"cancel_after_begin",
	"cancel_after_first_response",
	"client_streaming",
	"custom_metadata",
	"empty_stream",
	"empty_unary",
	"large_unary",
	"ping_pong",
	"server_streaming",
	"special_status_message",
	"status_code_and_message",
	"timeout_on_sleeping_server",
	"unimplemented_method",
	"unimplemented_service",
}

func TestNames() []string {
	return defaultTestCases
}

func RunTests(conn *grpc.ClientConn, testNames []string) error {
	for _, name := range testNames {
		if err := RunTest(conn, name); err != nil {
			return err
		}
	}
	return nil
}

func RunTest(conn *grpc.ClientConn, testName string) error {
	tc := testpb.NewTestServiceClient(conn)

	switch testName {
	case "cancel_after_begin":
		return DoCancelAfterBegin(tc)
	case "cancel_after_first_response":
		return DoCancelAfterFirstResponse(tc)
	case "client_streaming":
		return DoClientStreaming(tc)
	case "custom_metadata":
		return DoCustomMetadata(tc)
	case "empty_stream":
		return DoEmptyStream(tc)
	case "empty_unary":
		return DoEmptyUnaryCall(tc)
	case "large_unary":
		return DoLargeUnaryCall(tc)
	case "ping_pong":
		return DoPingPong(tc)
	case "server_streaming":
		return DoServerStreaming(tc)
	case "special_status_message":
		return DoSpecialStatusMessage(tc)
	case "status_code_and_message":
		return DoStatusCodeAndMessage(tc)
	case "timeout_on_sleeping_server":
		return DoTimeoutOnSleepingServer(tc)
	case "unimplemented_method":
		return DoUnimplementedMethod(conn)
	case "unimplemented_service":
		return DoUnimplementedService(testpb.NewUnimplementedServiceClient(conn))
	default:
		return fmt.Errorf("unknown test name")
	}
}
