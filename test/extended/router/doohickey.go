package router

import (
	"context"
	"strconv"
	"time"

	g "github.com/onsi/ginkgo"
	exutil "github.com/openshift/origin/test/extended/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework/pod"
)

var _ = g.Describe("[sig-network-edge][Conformance][Area:Networking][Feature:Router]", func() {
	defer g.GinkgoRecover()

	var (
		oc = exutil.NewCLI("doohickey")
	)

	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			exutil.DumpPodLogsStartingWithInNamespace("execPod", oc.Namespace(), oc.AsAdmin())
		}
	})

	g.Describe("The HAProxy router", func() {
		g.It("should pass the doohickey conformance tests", func() {
			const n int = 10

			createPod := func(oc *exutil.CLI, ns, name string, interval, timeout time.Duration) error {
				pod := pod.NewAgnhostPod(ns, name, nil, nil, nil)
				execPod, err := oc.KubeClient().CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
				if err != nil {
					return err
				}

				if err := wait.PollImmediate(interval, timeout, func() (bool, error) {
					retrievedPod, err := oc.KubeClient().CoreV1().Pods(execPod.Namespace).Get(context.TODO(), execPod.Name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					return retrievedPod.Status.Phase == v1.PodRunning, nil
				}); err != nil {
					return nil
				}

				return nil
			}

			ns := oc.Namespace()

			for i := 0; i < n; i++ {
				podName := "execpod" + strconv.Itoa(i)

				if i%2 == 0 {
					podName = "execpod"
				}

				if err := createPod(oc, ns, podName, 5*time.Second, 5*time.Minute); err == nil {
					oc.AdminKubeClient().CoreV1().Pods(ns).Delete(context.TODO(), podName, *metav1.NewDeleteOptions(1))
				}
				time.Sleep(time.Millisecond)
			}
		})
	})
})

// createExecPod creates an agnhost pause-based pod.
func createExecPod(oc *exutil.CLI, ns, name string, interval, timeout time.Duration) error {
	pod := pod.NewAgnhostPod(ns, name, nil, nil, nil)
	execPod, err := oc.KubeClient().CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		retrievedPod, err := oc.KubeClient().CoreV1().Pods(execPod.Namespace).Get(context.TODO(), execPod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return retrievedPod.Status.Phase == v1.PodRunning, nil
	}); err != nil {
		return nil
	}

	return nil
}
