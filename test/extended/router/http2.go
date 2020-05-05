package router

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	"golang.org/x/net/http2"

	routev1 "github.com/openshift/api/route/v1"
	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	exutil "github.com/openshift/origin/test/extended/util"
	"github.com/openshift/origin/test/extended/util/url"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	// http2TestTimeout is the timeout value for the
	// internal tests.
	http2TestTimeout = 2 * time.Minute

	// http2TestCaseIterations is the number of times each test
	// case should be invoked.
	http2TestCaseIterations = 5
)

var _ = g.Describe("[sig-network-edge][Conformance][Area:Networking][Feature:Router]", func() {
	defer g.GinkgoRecover()

	var (
		configPath = exutil.FixturePath("testdata", "router", "router-http2.yaml")
		oc         = exutil.NewCLI("router-http2")
	)

	// this hook must be registered before the framework namespace teardown
	// hook
	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			client := routeclientset.NewForConfigOrDie(oc.AdminConfig()).RouteV1().Routes(oc.KubeFramework().Namespace.Name)
			if routes, _ := client.List(context.Background(), metav1.ListOptions{}); routes != nil {
				outputIngress(routes.Items...)
			}
			exutil.DumpPodLogsStartingWith("http2", oc)
		}
	})

	g.Describe("The HAProxy router", func() {
		g.It("should pass the http2 tests", func() {
			g.By(fmt.Sprintf("creating test fixture from a config file %q", configPath))
			err := oc.Run("new-app").Args("-f", configPath).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			e2e.ExpectNoError(oc.KubeFramework().WaitForPodRunning("http2"))

			for _, route := range []routev1.TLSTerminationType{
				routev1.TLSTerminationEdge,
				routev1.TLSTerminationReencrypt,
				routev1.TLSTerminationPassthrough,
			} {
				g.By("verifying accessing the host returns a 200 status code")
				urlTester := url.NewTester(oc.AdminKubeClient(), oc.KubeFramework().Namespace.Name).WithErrorPassthrough(true)
				defer urlTester.Close()
				hostname := getHostnameForRoute(oc, fmt.Sprintf("http2-%s", route))
				urlTester.Within(30*time.Second, url.Expect("GET", "https://"+hostname).Through(hostname).SkipTLSVerification().HasStatusCode(200))
			}

			http2ExecOutOfClusterRouteTests(oc, http2TestCaseIterations)
		})
	})
})

// http2ExecOutOfClusterRouteTests run http2 tests using
// routes and from outside of the test cluster.
func http2ExecOutOfClusterRouteTests(oc *exutil.CLI, iterations int) {
	for _, route := range []routev1.TLSTerminationType{
		routev1.TLSTerminationEdge,
		routev1.TLSTerminationReencrypt,
		routev1.TLSTerminationPassthrough,
	} {
		// client := &http.Client{
		// 	Transport: &http.Transport{
		// 		TLSClientConfig: &tls.Config{
		// 			InsecureSkipVerify: true,
		// 		},
		// 	},
		// 	Timeout: http2TestTimeout,
		// }

		client := &http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			Timeout: http2TestTimeout,
		}

		hostname := getHostnameForRoute(oc, fmt.Sprintf("http2-%s", route))
		resp, err := client.Get("https://" + hostname)
		o.Expect(err).NotTo(o.HaveOccurred())
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)
	}
}

// http2ExecClientShellCmd runs the given cmd in the context of
// the "client-shell" container in the "http2" POD.
func http2ExecClientShellCmd(ns string, timeout time.Duration, cmd string) (string, error) {
	return e2e.RunKubectl(ns, "exec", fmt.Sprintf("--namespace=%v", ns), "http2", "-c", "client-shell", "--", "/bin/sh", "-x", "-c", fmt.Sprintf("timeout %v %s", timeout.Seconds(), cmd))
}
