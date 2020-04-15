package grpc_interop

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/interop"
	testpb "google.golang.org/grpc/interop/grpc_testing"
)

type testFn func(tc testpb.TestServiceClient, args ...grpc.CallOption)

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

func runInteropTest(tc testpb.TestServiceClient, conn *grpc.ClientConn, testNames []string) {
	for i, name := range testNames {
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
		grpclog.Infof("[#%v/%v] Test %q DONE\n", i+1, len(testNames), name)
	}
}
