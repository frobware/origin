package router

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	"golang.org/x/net/http2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"

	operatorv1 "github.com/openshift/api/operator/v1"
	v1 "github.com/openshift/api/route/v1"
	exutil "github.com/openshift/origin/test/extended/util"
)

const (
	// http2ClientGetTimeout specifies the time limit for requests
	// made by the HTTP Client.
	http2ClientTimeout = 1 * time.Minute
)

const cert = `
-----BEGIN CERTIFICATE-----
MIIBgTCCASagAwIBAgIRAO2UsHGM2j+IZfxSG0KSSH0wCgYIKoZIzj0EAwIwJDEQ
MA4GA1UEChMHUmVkIEhhdDEQMA4GA1UEAxMHUm9vdCBDQTAgFw0yMDA1MTExMDU2
NThaGA8yMTIwMDQxNzEwNTY1OFowJjEQMA4GA1UEChMHUmVkIEhhdDESMBAGA1UE
AwwJdGVzdF9jZXJ0MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEjCp5jERwCaBp
jhoaOxgD1DDaPwKkWu+a8mJLn9Cn9+LcSub05zPaMQy0a+FkvZSyTA2y8a8/IMM1
f2MQC6bATqM1MDMwDgYDVR0PAQH/BAQDAgWgMBMGA1UdJQQMMAoGCCsGAQUFBwMB
MAwGA1UdEwEB/wQCMAAwCgYIKoZIzj0EAwIDSQAwRgIhALwL+vTFS37a/R6RMeNN
fKKM6dOZeTSIVk6eGen6ZmZmAiEAhdLQwSu7ev/GrGwINF1rraoyrgiq4mFdPwHa
TctfSDo=
-----END CERTIFICATE-----
`

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
		shardConfigPath = exutil.FixturePath("testdata", "router", "router-http2-shard.yaml")
		configPath      = exutil.FixturePath("testdata", "router", "router-http2.yaml")
		oc              = exutil.NewCLI("router-http2")

		routerShardConfig string
	)

	// this hook must be registered before the framework namespace teardown
	// hook
	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			// exutil.DumpPodLogsStartingWith("http2", oc)
			// exutil.DumpPodLogsStartingWithInNamespace("router", "openshift-ingress", oc.AsAdmin())
		}
		if len(routerShardConfig) > 0 {
			oc.AsAdmin().Run("delete").Args("-n", "openshift-ingress-operator", "-f", routerShardConfig).Execute()
		}
	})

	g.Describe("The HAProxy router", func() {
		g.It("should pass the http2 tests", func() {
			g.By(fmt.Sprintf("creating test fixture from a config file %q", configPath))
			err := oc.Run("new-app").Args("--dry-run", "-f", configPath, "-p", "TLS_CRT="+fmt.Sprintf("%s", strings.ReplaceAll(cert[1:], "\n", `\n`))).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			err = oc.AsAdmin().Run("label").Args("namespace", oc.Namespace(), "type="+oc.Namespace()).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			e2e.ExpectNoError(e2epod.WaitForPodRunningInNamespaceSlow(oc.KubeClient(), "http2", oc.Namespace()), "http2 backend server pod not running")

			defaultDomain, err := getDefaultIngressClusterDomainName(oc, 5*time.Minute)
			o.Expect(err).NotTo(o.HaveOccurred(), "failed to find default domain name")

			shardDomain := oc.Namespace() + "." + defaultDomain

			// roll out ingresscontroller for this test only
			g.By(fmt.Sprintf("creating router shard %q from a config file %q", oc.Namespace(), shardConfigPath))
			routerShardConfig, err = oc.AsAdmin().Run("process").Args("-f", shardConfigPath, "-p", fmt.Sprintf("NAME=%v", oc.Namespace()), "NAMESPACE=openshift-ingress-operator", "DOMAIN="+shardDomain, "NAMESPACE_SELECTOR="+oc.Namespace()).OutputToFile("http2config.json")
			o.Expect(err).NotTo(o.HaveOccurred(), "ingresscontroller conditions not met")
			err = oc.AsAdmin().Run("create").Args("-f", routerShardConfig, "--namespace=openshift-ingress-operator").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			err = waitForIngressControllerCondition(oc, 15*time.Minute, types.NamespacedName{Namespace: "openshift-ingress-operator", Name: oc.Namespace()}, ingressControllerNonDefaultAvailableConditions...)
			o.Expect(err).NotTo(o.HaveOccurred(), "ingresscontroller conditions not met")

			testCases := []struct {
				route             string
				frontendProto     string
				backendProto      string
				statusCode        int
				useHTTP2Transport bool
				expectedGetError  string
			}{{
				route:             "http2-custom-cert-edge",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				route:             "http2-custom-cert-reencrypt",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				route:             "http2-passthrough",
				frontendProto:     "HTTP/2.0",
				backendProto:      "HTTP/2.0",
				statusCode:        http.StatusOK,
				useHTTP2Transport: true,
			}, {
				route:             "http2-default-cert-edge",
				useHTTP2Transport: true,
				expectedGetError:  `http2: unexpected ALPN protocol ""; want "h2"`,
			}, {
				route:             "http2-default-cert-reencrypt",
				useHTTP2Transport: true,
				expectedGetError:  `http2: unexpected ALPN protocol ""; want "h2"`,
			}, {
				route:             "http2-custom-cert-edge",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				route:             "http2-custom-cert-reencrypt",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/2.0", // reencrypt always has backend ALPN enabled
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				route:             "http2-passthrough",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				route:             "http2-default-cert-edge",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/1.1",
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}, {
				route:             "http2-default-cert-reencrypt",
				frontendProto:     "HTTP/1.1",
				backendProto:      "HTTP/2.0", // reencrypt always has backend ALPN enabled
				statusCode:        http.StatusOK,
				useHTTP2Transport: false,
			}}

			// Recreate routes with new shard namespace selector and host
			for _, tc := range testCases {
				host := tc.route + "." + shardDomain
				_, err := recreateRouteWithHostAndLabels(oc, tc.route, host, map[string]string{"type": oc.Namespace()}, 5*time.Minute)
				o.Expect(err).NotTo(o.HaveOccurred())
				e2e.Logf("recreated route %q with host=%q", tc.route, host)
			}

			// If we cannot resolve then we're not going
			// to make a connection, so assert that lookup
			// succeeds for each route.
			for _, tc := range testCases {
				err := wait.PollImmediate(time.Second, 15*time.Minute, func() (bool, error) {
					host := tc.route + "." + shardDomain
					addrs, err := net.LookupHost(host)
					if err != nil {
						e2e.Logf("host lookup error: %v, retrying...", err)
						return false, nil
					}
					e2e.Logf("host %q now resolves as %+v", host, addrs)
					return true, nil
				})
				o.Expect(err).NotTo(o.HaveOccurred())
			}

			for i, tc := range testCases {
				host := tc.route + "." + shardDomain
				e2e.Logf("[test #%d/%d]: GET route: %s", i+1, len(testCases), host)

				var resp *http.Response

				o.Expect(wait.Poll(time.Second, 15*time.Minute, func() (bool, error) {
					host := tc.route + "." + shardDomain
					client := makeHTTPClient(tc.useHTTP2Transport, http2ClientTimeout)
					resp, err = client.Get("https://" + host)
					if err != nil && len(tc.expectedGetError) != 0 {
						errMatch := strings.Contains(err.Error(), tc.expectedGetError)
						if !errMatch {
							e2e.Logf("[test #%d/%d]: route: %s, GET error: %v, retrying...", i+1, len(testCases), host, err)
						}
						return errMatch, nil
					}
					if err != nil {
						e2e.Logf("[test #%d/%d]: route: %s, GET error: %v, retrying...", i+1, len(testCases), host, err)
						return false, nil // could be 503 if service not ready
					}
					if tc.statusCode == 0 {
						return false, nil
					}
					if resp.StatusCode != tc.statusCode {
						e2e.Logf("[test #%d/%d]: route: %s, expected status: %v, actual status: %v, retrying...", i+1, len(testCases), host, tc.statusCode, resp.StatusCode)
					}
					return resp.StatusCode == tc.statusCode, nil
				})).NotTo(o.HaveOccurred())

				if tc.expectedGetError != "" {
					continue
				}

				o.Expect(resp).ToNot(o.BeNil(), "response was nil")
				o.Expect(resp.StatusCode).To(o.Equal(tc.statusCode), "HTTP response code not matched")
				o.Expect(resp.Proto).To(o.Equal(tc.frontendProto), "protocol not matched")
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

func recreateRouteWithHostAndLabels(oc *exutil.CLI, routeName, host string, labels map[string]string, timeout time.Duration) (*v1.Route, error) {
	var newRoute *v1.Route

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		existingRoute, err := oc.RouteClient().RouteV1().Routes(oc.Namespace()).Get(context.Background(), routeName, metav1.GetOptions{})
		if err != nil {
			e2e.Logf("Error getting route %q: %v, retrying...", routeName, err)
			return false, err
		}

		if err := oc.RouteClient().RouteV1().Routes(oc.Namespace()).Delete(context.Background(), routeName, metav1.DeleteOptions{}); err != nil {
			e2e.Logf("Error deleting route %q: %v", routeName, err)
			return true, err
		}

		newRoute = &v1.Route{}

		//  Type
		newRoute.Kind = existingRoute.Kind
		newRoute.APIVersion = existingRoute.APIVersion

		// Metadata
		newRoute.Name = existingRoute.Name
		newRoute.Namespace = existingRoute.Namespace
		newRoute.Labels = labels

		// Spec
		existingRoute.Spec.DeepCopyInto(&newRoute.Spec)
		newRoute.Spec.Host = host

		e2e.Logf("updating route %q with host=%q", routeName, host)
		newRoute, err = oc.RouteClient().RouteV1().Routes(oc.Namespace()).Create(context.Background(), newRoute, metav1.CreateOptions{})
		if err != nil {
			e2e.Logf("Error creating route %q with host=%q: %v", routeName, host, err)
			return true, err
		}
		return true, nil
	})

	return newRoute, err
}

var ingressControllerNonDefaultAvailableConditions = []operatorv1.OperatorCondition{
	{Type: operatorv1.IngressControllerAvailableConditionType, Status: operatorv1.ConditionTrue},
	{Type: operatorv1.LoadBalancerManagedIngressConditionType, Status: operatorv1.ConditionTrue},
	{Type: operatorv1.LoadBalancerReadyIngressConditionType, Status: operatorv1.ConditionTrue},
	{Type: operatorv1.DNSManagedIngressConditionType, Status: operatorv1.ConditionTrue},
	{Type: operatorv1.DNSReadyIngressConditionType, Status: operatorv1.ConditionTrue},
	{Type: "Admitted", Status: operatorv1.ConditionTrue},
}

func operatorConditionMap(conditions ...operatorv1.OperatorCondition) map[string]string {
	conds := map[string]string{}
	for _, cond := range conditions {
		conds[cond.Type] = string(cond.Status)
	}
	return conds
}

func conditionsMatchExpected(expected, actual map[string]string) bool {
	filtered := map[string]string{}
	for k := range actual {
		if _, comparable := expected[k]; comparable {
			filtered[k] = actual[k]
		}
	}
	return reflect.DeepEqual(expected, filtered)
}

func waitForIngressControllerCondition(oc *exutil.CLI, timeout time.Duration, name types.NamespacedName, conditions ...operatorv1.OperatorCondition) error {
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		ic, err := oc.AdminOperatorClient().OperatorV1().IngressControllers(name.Namespace).Get(context.Background(), name.Name, metav1.GetOptions{})
		if err != nil {
			e2e.Logf("failed to get ingresscontroller %s/%s: %v, retrying...", name.Namespace, name.Name, err)
			return false, nil
		}
		expected := operatorConditionMap(conditions...)
		current := operatorConditionMap(ic.Status.Conditions...)
		met := conditionsMatchExpected(expected, current)
		if !met {
			e2e.Logf("ingresscontroller %s/%s conditions not met; wanted %+v, got %+v, retrying...", name.Namespace, name.Name, expected, current)
		}
		return met, nil
	})
}
