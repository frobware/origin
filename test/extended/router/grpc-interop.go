package router

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	routev1 "github.com/openshift/api/route/v1"
	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	grpc_interop "github.com/openshift/origin/test/extended/router/grpc-interop"
	exutil "github.com/openshift/origin/test/extended/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const gRPCInteropTestTimeout = 1 * time.Minute

var _ = g.Describe("[sig-network-edge][Conformance][Area:Networking][Feature:Router]", func() {
	defer g.GinkgoRecover()

	var (
		configPath = exutil.FixturePath("testdata", "router", "router-grpc-interop.yaml")
		oc         = exutil.NewCLI("router-grpc")
	)

	// this hook must be registered before the framework namespace teardown
	// hook
	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			client := routeclientset.NewForConfigOrDie(oc.AdminConfig()).RouteV1().Routes(oc.KubeFramework().Namespace.Name)
			if routes, _ := client.List(context.Background(), metav1.ListOptions{}); routes != nil {
				outputIngress(routes.Items...)
			}
			exutil.DumpPodLogsStartingWith("grpc", oc)
		}
	})

	g.Describe("The HAProxy router", func() {
		g.It("should pass the gRPC interoperability tests", func() {
			g.By(fmt.Sprintf("creating routes from a config file %q", configPath))

			err := oc.Run("new-app").Args("-f", configPath).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			ns := oc.KubeFramework().Namespace.Name
			e2e.ExpectNoError(oc.KubeFramework().WaitForPodRunning("grpc-interop"))

			makeClientCmd := func(host string, port int, useTLS bool, caCert string, insecure bool) string {
				cmd := fmt.Sprintf("/workdir/grpc-client -host %q -port %v", host, port)
				if useTLS {
					cmd = fmt.Sprintf("%s -tls", cmd)
				}
				if caCert != "" {
					cmd = fmt.Sprintf("%s -ca-cert %q", cmd, caCert)
				}
				if insecure {
					cmd = fmt.Sprintf("%s -insecure", cmd)
				}
				return cmd
			}

			// run gRPC interop tests against the internal
			// service host:port from a POD within the
			// test cluster.
			for _, tc := range []struct {
				port     int
				useTLS   bool
				caCert   string
				insecure bool
			}{{
				port:   1110, // h2c
				useTLS: false,
			}, {
				port:   8443, // h2
				useTLS: true,
				caCert: "/etc/service-ca/service-ca.crt",
			}} {
				testCases := grpc_interop.SortedTestNames()
				svc := fmt.Sprintf("grpc-interop.%s.svc", ns)
				cmd := makeClientCmd(svc, tc.port, tc.useTLS, tc.caCert, tc.insecure) + " " + strings.Join(testCases, " ")
				e2e.Logf("Running gRPC interop test: %s", cmd)
				_, err := runInteropClientCmd(ns, gRPCInteropTestTimeout, cmd)
				o.Expect(err).NotTo(o.HaveOccurred())
			}

			// run gRPC interop tests via external routes
			// from outside of the test cluster.
			for _, route := range []routev1.TLSTerminationType{
				routev1.TLSTerminationEdge,
				routev1.TLSTerminationPassthrough,
				routev1.TLSTerminationReencrypt,
			} {
				testCases := grpc_interop.SortedTestNames()
				clientCfg := grpcClientConnConfig{
					host:     getHostnameForRoute(oc, fmt.Sprintf("grpc-interop-%s", route)),
					port:     443,
					useTLS:   true,
					insecure: true,
				}
				conn, err := grpcDial(clientCfg)
				o.Expect(err).NotTo(o.HaveOccurred())
				e2e.Logf("running gRPC interop tests %v against route %q:%v", testCases, clientCfg.host, clientCfg.port)
				grpc_interop.RunTests(conn, testCases)
				o.Expect(conn.Close()).NotTo(o.HaveOccurred())
			}
		})
	})
})

// runInteropCmd runs the given cmd in the context of the
// "client-shell" container in the "grpc-interop" POD.
func runInteropClientCmd(ns string, timeout time.Duration, cmd string) (string, error) {
	return e2e.RunKubectl(ns, "exec", fmt.Sprintf("--namespace=%v", ns), "grpc-interop", "-c", "client-shell", "--", "/bin/sh", "-x", "-c", fmt.Sprintf("timeout %v %s", timeout.Seconds(), cmd))
}

type grpcClientConnConfig struct {
	useTLS   bool
	certData []byte
	host     string
	port     int
	insecure bool
}

func grpcDial(cfg grpcClientConnConfig) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if cfg.useTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.insecure,
		}
		if len(cfg.certData) > 0 {
			rootCAs, err := x509.SystemCertPool()
			if err != nil {
				return nil, err
			}
			if rootCAs == nil {
				rootCAs = x509.NewCertPool()
			}
			if ok := rootCAs.AppendCertsFromPEM(cfg.certData); !ok {
				e2e.Logf("No certs appended, using system certs only")
			}
			tlsConfig.RootCAs = rootCAs
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	return grpc.Dial(net.JoinHostPort(cfg.host, strconv.Itoa(cfg.port)), append(opts, grpc.WithBlock())...)
}
