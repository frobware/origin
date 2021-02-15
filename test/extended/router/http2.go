package router

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/davecgh/go-spew/spew"
	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	"golang.org/x/net/http2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"

	v1 "github.com/openshift/api/route/v1"
	exutil "github.com/openshift/origin/test/extended/util"
)

const (
	// http2ClientGetTimeout specifies the time limit for requests
	// made by the HTTP Client.
	http2ClientTimeout = 1 * time.Minute

	tlsKey = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIFTS3l3jU0kF0cQ27MTn+a27Oz1CMr5gSnDQQwZ3BI0moAoGCCqGSM49
AwEHoUQDQgAE0XRkp541S/i1bBYEtl/xWsKHDtmuFu7X1rIQiRAU4SB43B52iDxJ
V5P5usDcYPghidR7m0hVj6s5iMskxwLr6g==
-----END EC PRIVATE KEY-----
`

	tlsCert = `
-----BEGIN CERTIFICATE-----
MIIBizCCATGgAwIBAgIQJN1GzcsQZOXK7xqks+2aijAKBggqhkjOPQQDAjAoMRQw
EgYDVQQKEwtDZXJ0IEdlbiBDbzEQMA4GA1UEAxMHUm9vdCBDQTAgFw0yMDA1MTgw
OTU3MjNaGA8yMTIwMDQyNDA5NTcyM1owLjEZMBcGA1UEChMQQ2VydCBHZW4gQ29t
cGFueTERMA8GA1UEAxMIdGVzdGNlcnQwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNC
AATRdGSnnjVL+LVsFgS2X/FawocO2a4W7tfWshCJEBThIHjcHnaIPElXk/m6wNxg
+CGJ1HubSFWPqzmIyyTHAuvqozUwMzAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAww
CgYIKwYBBQUHAwEwDAYDVR0TAQH/BAIwADAKBggqhkjOPQQDAgNIADBFAiBgjygk
ycGFHUK+b3hdvylD3ftaRT/muv+FKuuMqNCHsAIhAPW1owpVKlLHC/NaApFW/d0f
Gf2Mda0JAqYv0ehJUh7w
-----END CERTIFICATE-----
`
)

func makeHTTPClient(useHTTP2Transport bool, timeout time.Duration) *http.Client {
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
	}

	c := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
		},
	}

	if useHTTP2Transport {
		c.Transport = &http2.Transport{
			TLSClientConfig: &tlsConfig,
		}
	}

	return c
}

var _ = g.Describe("[sig-network-edge][Conformance][Area:Networking][Feature:Router]", func() {
	defer g.GinkgoRecover()

	var (
		shardConfigPath = exutil.FixturePath("testdata", "router", "router-h2shard.yaml")
		configPath      = exutil.FixturePath("testdata", "router", "router-http2.yaml")
		oc              = exutil.NewCLI("router-http2")
	)

	// this hook must be registered before the framework namespace teardown
	// hook
	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			//exutil.DumpPodLogsStartingWith("http2", oc)
			//exutil.DumpPodLogsStartingWithInNamespace("router", oc.Namespace(), oc.AsAdmin())
		}
	})

	g.Describe("The HAProxy router", func() {
		g.It("should pass the http2 tests", func() {
			g.By(fmt.Sprintf("creating test fixture from a config file %q", configPath))
			err := oc.Run("new-app").Args("-f", configPath).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			defaultDomain, err := getDefaultIngressClusterDomainName(oc, 5*time.Minute)
			o.Expect(err).NotTo(o.HaveOccurred(), "failed to find default domain name")

			http2Subdomain := strings.ReplaceAll(oc.Namespace(), "e2e-test-router-", "")
			o.Expect(http2Subdomain).ShouldNot(o.BeEmpty())
			http2Domain := http2Subdomain + "." + defaultDomain

			g.By(fmt.Sprintf("creating router shard %q from a config file %q", http2Subdomain, shardConfigPath))
			shardConfigFile, err := oc.AsAdmin().Run("process").Args("-f", shardConfigPath, "-p", fmt.Sprintf("NAME=%v", http2Subdomain), "NAMESPACE=openshift-ingress-operator", "DOMAIN="+http2Domain).OutputToFile("http2config.json")

			defer func() {
				// Always attemtp tp clean up ingresscontroller shard
				oc.AsAdmin().Run("delete").Args("-f", shardConfigFile, "--namespace=openshift-ingress-operator").Execute()
			}()

			err = oc.AsAdmin().Run("create").Args("-f", shardConfigFile, "--namespace=openshift-ingress-operator").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			e2e.ExpectNoError(e2epod.WaitForPodRunningInNamespaceSlow(oc.KubeClient(), "http2", oc.Namespace()), "text fixture pods not running")

			testCases := []struct {
				route             v1.Route
				routeName         string
				frontendProto     string
				backendProto      string
				statusCode        int
				useHTTP2Transport bool
				expectedGetError  string
			}{{
				route: v1.Route{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Route",
						APIVersion: "route.openshift.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "http2-custom-cert-edge",
						Namespace: oc.Namespace(),
					},
					Spec: v1.RouteSpec{
						To: v1.RouteTargetReference{
							Kind: "Service",
							Name: "http2",
						},
						Port: &v1.RoutePort{
							TargetPort: intstr.IntOrString{
								IntVal: 8080,
							},
						},
						TLS: &v1.TLSConfig{
							Termination: v1.TLSTerminationEdge,
							Certificate: tlsCert[1:],
							Key:         tlsKey[1:],
						},
						WildcardPolicy: v1.WildcardPolicyNone,
					},
				},
				routeName:         "http2-custom-cert-edge",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeName:         "http2-custom-cert-reencrypt",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeName:         "http2-passthrough",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				routeName:         "http2-default-cert-edge",
				useHTTP2Transport: true,
				expectedGetError:  `http2: unexpected ALPN protocol ""; want "h2"`,
			}, {
				routeName:         "http2-default-cert-reencrypt",
				useHTTP2Transport: true,
				expectedGetError:  `http2: unexpected ALPN protocol ""; want "h2"`,
			}, {
				routeName:         "http2-custom-cert-edge",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				routeName:         "http2-custom-cert-reencrypt",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/2.0", // reencrypt always has backend ALPN enabled
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				routeName:         "http2-passthrough",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				routeName:         "http2-default-cert-edge",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				routeName:         "http2-default-cert-reencrypt",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/2.0", // reencrypt always has backend ALPN enabled
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}}

			for i := range testCases {
				if i > 0 {
					break
				}

				route := &testCases[i].route
				route.Name = testCases[i].routeName
				route.Spec.Host = testCases[i].routeName + "." + http2Domain
				route.Labels = map[string]string{
					"type": http2Subdomain,
				}
				spew.Dump(route)
				_, err := oc.RouteClient().RouteV1().Routes(oc.Namespace()).Create(context.Background(), route.DeepCopy(), metav1.CreateOptions{})
				o.Expect(err).NotTo(o.HaveOccurred())
				e2e.Logf("creaated route %q with host=%q", route.Name, route.Spec.Host)
			}

			for i, tc := range testCases {
				if i > 0 {
					break
				}

				err := wait.PollImmediate(3*time.Second, 5*time.Minute, func() (bool, error) {
					addrs, err := net.LookupHost(tc.route.Spec.Host)
					if err != nil {
						e2e.Logf("lookup %q failed: %v, retrying...", tc.route.Spec.Host, err)
						return false, nil
					}
					e2e.Logf("host %q now resolves as %+v", tc.route.Spec.Host, addrs)
					return true, nil
				})
				o.Expect(err).NotTo(o.HaveOccurred())
			}

			for i, tc := range testCases {
				if i > 0 {
					break
				}

				testConfig := fmt.Sprintf("%+v", tc)
				e2e.Logf("[test #%d/%d]: config: %s", i+1, len(testCases), tc.routeName)

				var resp *http.Response

				o.Expect(wait.Poll(3*time.Second, 15*time.Minute, func() (bool, error) {
					client := makeHTTPClient(tc.useHTTP2Transport, http2ClientTimeout)
					resp, err = client.Get("https://" + tc.route.Spec.Host)
					if err != nil && len(tc.expectedGetError) != 0 {
						errMatch := strings.Contains(err.Error(), tc.expectedGetError)
						if !errMatch {
							e2e.Logf("[AAA test #%d/%d]: config: %s, GET error: %v", i+1, len(testCases), tc.routeName, err)
						}
						return errMatch, nil
					}
					if err != nil {
						e2e.Logf("[BBB test #%d/%d]: config: %s, GET error: %v, resp=%+v", i+1, len(testCases), tc.routeName, err, resp)
						return false, nil // could be 503 if service not ready
					}
					if tc.statusCode == 0 {
						return false, nil
					}
					e2e.Logf("[DDD test #%d/%d]: config: %s, expected status: %v, actual status: %v", i+1, len(testCases), tc.routeName, tc.statusCode, resp.StatusCode)
					return resp.StatusCode == tc.statusCode, nil
				})).NotTo(o.HaveOccurred())

				if tc.expectedGetError != "" {
					continue
				}

				o.Expect(resp).ToNot(o.BeNil(), "response was nil")
				o.Expect(resp.StatusCode).To(o.Equal(tc.statusCode), testConfig, "HTTP response code not matched")
				o.Expect(resp.Proto).To(o.Equal(tc.frontendProto), testConfig, "protocol not matched")
				body, err := ioutil.ReadAll(resp.Body)
				o.Expect(err).NotTo(o.HaveOccurred(), "failed to read the response body")
				o.Expect(string(body)).To(o.Equal(tc.backendProto), "response body content not matched")
				o.Expect(resp.Body.Close()).NotTo(o.HaveOccurred(), "failed to close response body")
			}
		})
	})
})

func getDefaultIngressClusterDomainName(oc *exutil.CLI, timeout time.Duration) (string, error) {
	var domain string

	if err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		ingress, err := oc.AdminConfigClient().ConfigV1().Ingresses().Get(context.TODO(), "cluster", metav1.GetOptions{})
		if err != nil {
			e2e.Logf("Get ingresses.config/cluster failed: %v, retrying...", err)
			return false, nil
		}
		domain = ingress.Spec.Domain
		return true, nil
	}); err != nil {
		return "", err
	}

	return domain, nil
}

func replaceRoute(oc *exutil.CLI, routeName, host string, timeout time.Duration) (*v1.Route, error) {
	var r *v1.Route

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		e2e.Logf("updating route %q with host=%q", routeName, host)
		route, err := oc.RouteClient().RouteV1().Routes(oc.Namespace()).Get(context.Background(), routeName, metav1.GetOptions{})
		if err != nil {
			e2e.Logf("Error getting route %q: %v, retrying...", routeName, err)
			return false, err
		}

		savedRoute := route.DeepCopy()

		if err := oc.RouteClient().RouteV1().Routes(oc.Namespace()).Delete(context.Background(), routeName, metav1.DeleteOptions{}); err != nil {
			e2e.Logf("Error deleting route %q: %v", routeName, err)
			return true, err
		}

		route.Namespace = savedRoute.Namespace
		savedRoute.Spec.Host = host
		savedRoute.Status = v1.RouteStatus{}

		if _, err = oc.RouteClient().RouteV1().Routes(oc.Namespace()).Create(context.Background(), savedRoute, metav1.CreateOptions{}); err != nil {
			e2e.Logf("Error creating route %q with host=%q: %v", routeName, host, err)
			return true, err
		}

		r = savedRoute
		return true, nil
	})

	return r, err
}
