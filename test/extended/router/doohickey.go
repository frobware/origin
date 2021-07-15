package router

import (
	"context"
	"fmt"
	"strconv"
	"time"

	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework/pod"
)

const podNamePrefix = "latency-"

var errPodCompleted = fmt.Errorf("pod ran to completion")

var _ = g.Describe("[sig-network-edge][Conformance][Area:Networking][Feature:Router]", func() {
	// defer g.GinkgoRecover()

	var (
		oc = exutil.NewCLI("doohickey")
	)

	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			// exutil.DumpPodLogsStartingWithInNamespace(podNamePrefix, oc.Namespace(), oc.AsAdmin())
		}
	})

	g.Describe("The HAProxy router", func() {
		g.It("should pass the doohickey conformance tests", func() {
			const n int = 10

			createPod := func(oc *exutil.CLI, ns, name string, interval, timeout time.Duration) error {
				pod := pod.NewExecPodSpec(ns, name, false)
				if _, err := oc.KubeClient().CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
					return err
				}
				return wait.PollImmediate(interval, timeout, func() (bool, error) {
					pod, err := oc.KubeClient().CoreV1().Pods(ns).Get(context.TODO(), name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					return pod.Status.Phase == v1.PodRunning, nil
				})
			}

			ns := oc.Namespace()

			for i := 0; i < n; i++ {
				podName := podNamePrefix + strconv.Itoa(i)

				if i%2 != 0 {
					podName = podNamePrefix
				}

				err := createPod(oc, ns, podName, 1*time.Second, 100*time.Millisecond)
				o.Expect(err).NotTo(o.HaveOccurred())
				// oc.AdminKubeClient().CoreV1().Pods(ns).Delete(context.TODO(), podName, *metav1.NewDeleteOptions(1))
				time.Sleep(time.Millisecond)
			}
		})
	})
})
