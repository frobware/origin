package router

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	"golang.org/x/net/http2"

	routev1 "github.com/openshift/api/route/v1"
	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	exutil "github.com/openshift/origin/test/extended/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	// http2ClientGetTimeout specifies the time limit for requests
	// made by the HTTP Client.
	http2ClientTimeout = 1 * time.Minute
)

func makeHTTPClient(useHTTP2Transport bool, timeout time.Duration) *http.Client {
	tls := tls.Config{
		InsecureSkipVerify: true,
	}

	c := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls,
		},
	}

	if useHTTP2Transport {
		c.Transport = &http2.Transport{
			TLSClientConfig: &tls,
		}
	}

	return c
}

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
			e2e.ExpectNoError(oc.KubeFramework().WaitForPodRunningSlow("http2"))

			for _, tc := range []struct {
				routeType         routev1.TLSTerminationType
				routeHostPrefix   string
				frontendProto     string
				backendProto      string
				expectedStatus    int
				useHTTP2Transport bool
			}{{
				routeType:         routev1.TLSTerminationEdge,
				routeHostPrefix:   "http2-custom-cert",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/1.1",
				expectedStatus:    http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeType:         routev1.TLSTerminationReencrypt,
				routeHostPrefix:   "http2-custom-cert",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				expectedStatus:    http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeType:         routev1.TLSTerminationPassthrough,
				routeHostPrefix:   "http2-custom-cert",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				expectedStatus:    http.StatusOK,
				useHTTP2Transport: true,
			}} {
				hostname := getHostnameForRoute(oc, fmt.Sprintf("%s-%s", tc.routeHostPrefix, tc.routeType))
				err := wait.Poll(time.Second, 30*time.Second, func() (bool, error) {
					client := makeHTTPClient(tc.useHTTP2Transport, http2ClientTimeout)
					url := "https://" + hostname
					resp, err := client.Get(url)
					if err != nil {
						e2e.Logf("client.Get(%q) failed: %v, retrying...", url, err)
						return false, nil
					}
					id := tc.routeHostPrefix + "-" + string(tc.routeType)
					defer resp.Body.Close()
					if !(resp.StatusCode == tc.expectedStatus && resp.Proto == tc.frontendProto) {
						e2e.Logf("[%s] Response(status: %v, protocol: %q), Expected(status: %v, protocol: %q), retrying...", id, resp.StatusCode, resp.Proto, tc.expectedStatus, tc.frontendProto)
						return false, nil
					}
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						e2e.Logf("[%s] ReadAll(resp.Body) failed: %q, retrying...", id, err)
						return false, nil
					}
					if !strings.Contains(string(body), tc.backendProto) {
						e2e.Logf("[%s] backend protocol not matched; expected %q, got %q, retrying...", tc.backendProto, string(body))
						return false, nil
					}
					return true, nil
				})
				o.Expect(err).NotTo(o.HaveOccurred())
			}
		})
	})
})
