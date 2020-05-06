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

			testCases := []struct {
				routeType         routev1.TLSTerminationType
				routeHostPrefix   string
				frontendProto     string
				backendProto      string
				statusCode        int
				useHTTP2Transport bool
				expectedGetError  string
			}{{
				routeType:         routev1.TLSTerminationEdge,
				routeHostPrefix:   "http2-custom-cert",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeType:         routev1.TLSTerminationReencrypt,
				routeHostPrefix:   "http2-custom-cert",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeType:         routev1.TLSTerminationPassthrough,
				routeHostPrefix:   "http2-custom-cert",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeType:         routev1.TLSTerminationEdge,
				routeHostPrefix:   "http2-default-cert",
				useHTTP2Transport: true,
				expectedGetError:  `http2: unexpected ALPN protocol ""; want "h2"`,
			}, {
				routeType:         routev1.TLSTerminationReencrypt,
				routeHostPrefix:   "http2-default-cert",
				useHTTP2Transport: true,
				expectedGetError:  `http2: unexpected ALPN protocol ""; want "h2"`,
			}, {
				routeType:         routev1.TLSTerminationPassthrough,
				routeHostPrefix:   "http2-default-cert",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}}

			for i, tc := range testCases {
				testConfig := fmt.Sprintf("%+v", tc)
				e2e.Logf("[test #%d/%d]: config: %s", i+1, len(testCases), testConfig)

				var resp *http.Response
				client := makeHTTPClient(tc.useHTTP2Transport, http2ClientTimeout)

				o.Expect(wait.Poll(time.Second, 30*time.Second, func() (bool, error) {
					var err error
					url := "https://" + getHostnameForRoute(oc, fmt.Sprintf("%s-%s", tc.routeHostPrefix, tc.routeType))
					resp, err = client.Get(url)
					if err != nil && tc.expectedGetError != "" {
						errMatch := strings.Contains(err.Error(), tc.expectedGetError)
						if !errMatch {
							e2e.Logf("[test #%d/%d]: config: %s, GET error: %v", i+1, len(testCases), testConfig, err)
						}
						return errMatch, nil
					}
					if err != nil {
						e2e.Logf("[test #%d/%d]: config: %s, GET error: %v", i+1, len(testCases), testConfig, tc.statusCode, resp.StatusCode)
						return false, nil
					}
					if resp.StatusCode != tc.statusCode {
						e2e.Logf("[test #%d/%d]: config: %s, expected status: %v, actual status: %v", i+1, len(testCases), testConfig, tc.statusCode, resp.StatusCode)
					}
					return resp.StatusCode == tc.statusCode, nil
				})).NotTo(o.HaveOccurred())

				if tc.expectedGetError != "" {
					continue
				}

				defer resp.Body.Close()
				o.Expect(resp.StatusCode).To(o.Equal(tc.statusCode), testConfig)
				o.Expect(resp.Proto).To(o.Equal(tc.frontendProto), testConfig)
				body, err := ioutil.ReadAll(resp.Body)
				o.Expect(err).NotTo(o.HaveOccurred(), testConfig)
				o.Expect(string(body)).To(o.Equal(tc.backendProto), testConfig)
			}
		})
	})
})
