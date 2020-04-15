package grpc_interop

import (
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

func RunTest(conn *grpc.ClientConn, testName string) error {
	switch testName {
	case "cancel_after_begin":
		return DoCancelAfterBegin(conn)
	case "cancel_after_first_response":
		return DoCancelAfterFirstResponse(conn)
	case "client_streaming":
		return DoClientStreaming(conn)
	case "custom_metadata":
		return DoCustomMetadata(conn)
	case "empty_stream":
		return DoEmptyStream(conn)
	case "empty_unary":
		return DoEmptyUnaryCall(conn)
	case "large_unary":
		return DoLargeUnaryCall(conn)
	case "ping_pong":
		return DoPingPong(conn)
	case "server_streaming":
		return DoServerStreaming(conn)
	case "special_status_message":
		return DoSpecialStatusMessage(conn)
	case "status_code_and_message":
		return DoStatusCodeAndMessage(conn)
	case "timeout_on_sleeping_server":
		return DoTimeoutOnSleepingServer(conn)
	case "unimplemented_method":
		return DoUnimplementedMethod(conn)
	case "unimplemented_service":
		return DoUnimplementedService(testpb.NewTestServiceClient(conn))
	default:
		return new.Error("unknown test name")
	}
}
