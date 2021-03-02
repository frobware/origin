package router

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
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
	exutil "github.com/openshift/origin/test/extended/util"

	"github.com/openshift/origin/test/extended/router/certgen"
)

const (
	// http2ClientGetTimeout specifies the time limit for requests
	// made by the HTTP Client.
	http2ClientTimeout = 1 * time.Minute
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
		http2ServiceConfigPath = exutil.FixturePath("testdata", "router", "router-http2.yaml")
		http2RoutesConfigPath  = exutil.FixturePath("testdata", "router", "router-http2-routes.yaml")
		shardConfigPath        = exutil.FixturePath("testdata", "router", "router-http2-shard.yaml")
		oc                     = exutil.NewCLI("router-http2")

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
			defaultDomain, err := getDefaultIngressClusterDomainName(oc, 5*time.Minute)
			o.Expect(err).NotTo(o.HaveOccurred(), "failed to find default domain name")

			srcTarGz, err := makeCompressedTarArchive([]string{"./test/extended/router/http2/cluster/server/server.go"})
			o.Expect(err).NotTo(o.HaveOccurred())
			base64SrcTarGz := strings.Join(split(base64.StdEncoding.EncodeToString(srcTarGz), 76), "\n")

			g.By(fmt.Sprintf("creating service from a config file %q", http2ServiceConfigPath))
			err = oc.Run("new-app").Args("-f", http2ServiceConfigPath, "-p", "BASE64_SRC_TGZ="+base64SrcTarGz).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			e2e.ExpectNoError(e2epod.WaitForPodRunningInNamespaceSlow(oc.KubeClient(), "http2", oc.Namespace()), "http2 backend server pod not running")

			// certificate start and end time are very
			// lenient to avoid any clock drift between
			// between the test machine and the cluster
			// under test.
			notBefore := time.Now().Add(-24 * time.Hour)
			notAfter := time.Now().Add(24 * time.Hour)

			// Generate crt/key for routes that need them.
			_, tlsCrtData, tlsPrivateKey, err := certgen.GenerateKeyPair(notBefore, notAfter)
			o.Expect(err).NotTo(o.HaveOccurred())

			derKey, err := certgen.MarshalPrivateKeyToDERFormat(tlsPrivateKey)
			o.Expect(err).NotTo(o.HaveOccurred())

			pemCrt, err := certgen.MarshalCertToPEMString(tlsCrtData)
			o.Expect(err).NotTo(o.HaveOccurred())

			shardedDomain := oc.Namespace() + "." + defaultDomain

			g.By(fmt.Sprintf("creating routes from a config file %q", http2RoutesConfigPath))
			err = oc.Run("new-app").Args("-f", http2RoutesConfigPath,
				"-p", "DOMAIN="+shardedDomain,
				"-p", "TLS_CRT="+pemCrt,
				"-p", "TLS_KEY="+derKey,
				"-p", "TYPE="+oc.Namespace()).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			// Roll out ingresscontroller dedicated to this test only.
			g.By(fmt.Sprintf("creating router shard %q from a config file %q", oc.Namespace(), shardConfigPath))

			routerShardConfig, err = oc.AsAdmin().Run("process").Args("-f", shardConfigPath, "-p", fmt.Sprintf("NAME=%v", oc.Namespace()), "NAMESPACE=openshift-ingress-operator", "DOMAIN="+shardedDomain, "TYPE="+oc.Namespace()).OutputToFile("http2config.json")
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

			// If we cannot resolve then we're not going
			// to make a connection, so assert that lookup
			// succeeds for each route.
			for _, tc := range testCases {
				err := wait.PollImmediate(3*time.Second, 15*time.Minute, func() (bool, error) {
					host := tc.route + "." + shardedDomain
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

			// Label the namespace with type=<namespace>
			// as the router shard will use a namespace
			// selector.
			err = oc.AsAdmin().Run("label").Args("namespace", oc.Namespace(), "type="+oc.Namespace()).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			for i, tc := range testCases {
				host := tc.route + "." + shardedDomain
				e2e.Logf("[test #%d/%d]: GET route: %s", i+1, len(testCases), host)

				var resp *http.Response

				o.Expect(wait.Poll(time.Second, 15*time.Minute, func() (bool, error) {
					host := tc.route + "." + shardedDomain
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
	return wait.PollImmediate(3*time.Second, timeout, func() (bool, error) {
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

// readFile reads all data from filename, or fatally fails if an error
// occurs.
func readFile(filename string) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %v", filename, err)
	}
	return data, nil
}

func addPrefix(lines []string, prefix string) []string {
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return lines
}

// split string into chunks limited in length by size.
// Note: assumes 1:1 mapping between bytes/chars (i.e., non-UTF).
func split(s string, size int) []string {
	var chunks []string

	for len(s) > 0 {
		if len(s) < size {
			size = len(s)
		}
		chunks, s = append(chunks, s[:size]), s[size:]
	}

	return chunks
}

func makeCompressedTarArchive(filenames []string) ([]byte, error) {
	buf := new(bytes.Buffer)

	gz, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("Error: gzip.NewWriterLevel(): %v", err)
	}

	tw := tar.NewWriter(gz)

	for _, filename := range filenames {
		fi, err := os.Stat(filename)
		if err != nil {
			return nil, fmt.Errorf("Error: failed to stat %q: %v", filename, err)
		}

		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return nil, fmt.Errorf("Error: failed to create tar header for %q: %v", filename, err)

		}

		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		data, err := readFile(filename)
		if err != nil {
			return nil, err
		}

		if _, err := tw.Write(data); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
