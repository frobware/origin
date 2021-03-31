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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"

	configv1 "github.com/openshift/api/config/v1"
	exutil "github.com/openshift/origin/test/extended/util"

	"github.com/openshift/origin/test/extended/router/certgen"
	"github.com/openshift/origin/test/extended/router/shard"
)

const (
	// http2ClientGetTimeout specifies the time limit for requests
	// made by the HTTP Client.
	http2ClientTimeout = 1 * time.Minute

	ingressOperatorNamespace = "openshift-ingress-operator"
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
		http2ServiceConfigPath     = exutil.FixturePath("testdata", "router", "router-http2.yaml")
		http2RoutesConfigPath      = exutil.FixturePath("testdata", "router", "router-http2-routes.yaml")
		http2RouterShardConfigPath = exutil.FixturePath("testdata", "router", "router-shard.yaml")

		oc = exutil.NewCLI("router-http2")

		shardConfigPath string // computed
	)

	// this hook must be registered before the framework namespace teardown
	// hook
	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			exutil.DumpPodLogsStartingWith("http2", oc)
			exutil.DumpPodLogsStartingWithInNamespace("router", "openshift-ingress", oc.AsAdmin())
		}
		if len(shardConfigPath) > 0 {
			if err := oc.AsAdmin().Run("delete").Args("-n", ingressOperatorNamespace, "-f", shardConfigPath).Execute(); err != nil {
				e2e.Logf("deleting ingress controller failed: %v\n", err)
			}
		}
	})

	g.Describe("The HAProxy router", func() {
		g.It("should pass the http2 tests", func() {
			infra, err := oc.AdminConfigClient().ConfigV1().Infrastructures().Get(context.Background(), "cluster", metav1.GetOptions{})
			o.Expect(err).NotTo(o.HaveOccurred(), "failed to get cluster-wide infrastructure")
			switch infra.Status.PlatformStatus.Type {
			case configv1.OvirtPlatformType, configv1.KubevirtPlatformType, configv1.LibvirtPlatformType, configv1.VSpherePlatformType, configv1.BareMetalPlatformType, configv1.NonePlatformType:
				g.Skip("Skip on platforms where the default router is not exposed by a load balancer service.")
			}

			defaultDomain, err := getDefaultIngressClusterDomainName(oc, 5*time.Minute)
			o.Expect(err).NotTo(o.HaveOccurred(), "failed to find default domain name")

			image, err := findCanaryImage(oc)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("creating service")
			err = oc.Run("new-app").Args("-f", http2ServiceConfigPath, "-p", "IMAGE="+image).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			err = e2epod.WaitForPodRunningInNamespaceSlow(oc.KubeClient(), "http2", oc.Namespace())
			o.Expect(err).NotTo(o.HaveOccurred())

			// certificate start and end time are very
			// lenient to avoid any clock drift between
			// the test machine and the cluster under
			// test.
			notBefore := time.Now().Add(-24 * time.Hour)
			notAfter := time.Now().Add(24 * time.Hour)

			// Generate crt/key for routes that need them.
			_, tlsCrtData, tlsPrivateKey, err := certgen.GenerateKeyPair(notBefore, notAfter)
			o.Expect(err).NotTo(o.HaveOccurred())

			derKey, err := certgen.MarshalPrivateKeyToDERFormat(tlsPrivateKey)
			o.Expect(err).NotTo(o.HaveOccurred())

			pemCrt, err := certgen.MarshalCertToPEMString(tlsCrtData)
			o.Expect(err).NotTo(o.HaveOccurred())

			shardFQDN := oc.Namespace() + "." + defaultDomain

			// The new router shard is using a namespace
			// selector so label this test namespace to
			// match.
			err = oc.AsAdmin().Run("label").Args("namespace", oc.Namespace(), "type="+oc.Namespace()).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("creating routes")
			err = oc.Run("new-app").Args("-f", http2RoutesConfigPath,
				"-p", "DOMAIN="+shardFQDN,
				"-p", "TLS_CRT="+pemCrt,
				"-p", "TLS_KEY="+derKey,
				"-p", "TYPE="+oc.Namespace()).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("creating a test-specific router shard")
			shardConfigPath, err = shard.DeployNewRouterShard(oc, 15*time.Minute, shard.Config{
				FixturePath: http2RouterShardConfigPath,
				Name:        oc.Namespace(),
				Domain:      shardFQDN,
				Type:        oc.Namespace(),
			})
			o.Expect(err).NotTo(o.HaveOccurred(), "new ingresscontroller did not rollout")

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

			for i, tc := range testCases {
				var resp *http.Response

				client := makeHTTPClient(tc.useHTTP2Transport, http2ClientTimeout)

				o.Expect(wait.Poll(1*time.Second, 5*time.Minute, func() (bool, error) {
					host := tc.route + "." + shardFQDN
					e2e.Logf("[test #%d/%d]: GET route: %s", i+1, len(testCases), host)
					resp, err = client.Get("https://" + host)
					if err != nil && len(tc.expectedGetError) != 0 {
						errMatch := strings.Contains(err.Error(), tc.expectedGetError)
						if !errMatch {
							// We can hit the "default" ingress controller
							// so wait for the new one to be fully
							// registered and serving.
							e2e.Logf("[test #%d/%d]: route: %s, GET error: %v, resp: %+v), retrying...", i+1, len(testCases), host, err, resp)
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
				o.Expect(string(body)).To(o.Equal(tc.backendProto), fmt.Sprintf("response body content not matched: got %q, want %q", string(body), tc.backendProto))
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

func findCanaryImage(oc *exutil.CLI) (string, error) {
	o, err := oc.AdminConfigClient().ConfigV1().ClusterOperators().Get(context.Background(), "ingress", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, v := range o.Status.Versions {
		if v.Name == "canary-server" {
			return v.Version, nil
		}
	}
	return "", fmt.Errorf("expected to find canary-server version on clusteroperators/ingress")
}
