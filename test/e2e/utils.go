package e2e

import (
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"testing"
	"time"
)

func WaitForDeployment(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, replicas int, retryInterval, timeout time.Duration) error {
	return e2eutil.WaitForDeployment(t, kubeclient, namespace, name, replicas, retryInterval, timeout)
}

func waitForPod(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		pod, err := kubeclient.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if pod.Status.Phase == v1.PodRunning {
			return true, nil
		}

		return false, nil
	})

	return err
}

func WaitForReplicationController(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, replicas int, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {

		rc, err := kubeclient.Core().ReplicationControllers(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s rc\n", name)
				return false, nil
			}
			return false, err
		}

		if int(rc.Status.AvailableReplicas) == replicas {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s rc (%d/%d)\n", name, rc.Status.AvailableReplicas, replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Replication controller '%s' available (%d/%d)\n", name, replicas, replicas)
	return nil
}
