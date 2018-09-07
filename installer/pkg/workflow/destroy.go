package workflow

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

const (
	machineSetNamespace = "openshift-cluster-api"
	workerMachineSet    = "worker"
)

// DestroyWorkflow creates new instances of the 'destroy' workflow,
// responsible for running the actions required to remove resources
// of an existing cluster and clean up any remaining artefacts.
func DestroyWorkflow(clusterDir string) Workflow {
	return Workflow{
		metadata: metadata{clusterDir: clusterDir},
		steps: []step{
			readClusterConfigStep,
			generateTerraformVariablesStep,
			destroyBootstrapStep,
			destroyWorkersStep,
			destroyInfraStep,
			destroyAssetsStep,
		},
	}
}

func destroyAssetsStep(m *metadata) error {
	return runDestroyStep(m, assetsStep)
}

func destroyInfraStep(m *metadata) error {
	return runDestroyStep(m, infraStep)
}

func destroyBootstrapStep(m *metadata) error {
	return runDestroyStep(m, bootstrapStep)
}

func destroyWorkersStep(m *metadata) error {
	kubeconfig := filepath.Join(m.clusterDir, generatedPath, "auth", "kubeconfig")

	// TODO(bison): Should any of these actually be fatal errors?
	client, err := buildClusterClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("Failed to build cluster-api client: %v", err)
	}

	if err := scaleDownWorkers(client); err != nil {
		return fmt.Errorf("Failed to scale worker MachineSet: %v", err)
	}

	if err := waitForWorkers(client); err != nil {
		return fmt.Errorf("Worker MachineSet failed to scale down: %v", err)
	}

	if err := deleteWorkerMachineSet(client); err != nil {
		return fmt.Errorf("Failed to delete worker MachineSet: %v", err)
	}

	return nil
}

func scaleDownWorkers(client *clientset.Clientset) error {
	// Unfortunately, MachineSets don't yet support the scale
	// subresource.  So we have to patch the object to set the
	// replicas to zero.
	patch := []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value uint32 `json:"value"`
	}{{
		Op:    "replace",
		Path:  "/spec/replicas",
		Value: 0,
	}}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = client.ClusterV1alpha1().
		MachineSets(machineSetNamespace).
		Patch(workerMachineSet, types.JSONPatchType, patchBytes)

	return err
}

func waitForWorkers(client *clientset.Clientset) error {
	interval := 3 * time.Second
	timeout := 60 * time.Second

	log.Info("Waiting for worker MachineSet to scale down...")

	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		machineSet, err := client.ClusterV1alpha1().
			MachineSets(machineSetNamespace).
			Get(workerMachineSet, v1.GetOptions{})

		if err != nil {
			return false, err
		}

		if machineSet.Status.Replicas > 0 {
			return false, nil
		}

		return true, nil
	})

	return err
}

func deleteWorkerMachineSet(client *clientset.Clientset) error {
	return client.ClusterV1alpha1().
		MachineSets(machineSetNamespace).
		Delete(workerMachineSet, &v1.DeleteOptions{})
}

func runDestroyStep(m *metadata, step string, extraArgs ...string) error {
	if !hasStateFile(m.clusterDir, step) {
		// there is no statefile, therefore nothing to destroy for this step
		return nil
	}
	templateDir, err := findStepTemplates(step, m.cluster.Platform)
	if err != nil {
		return err
	}

	return tfDestroy(m.clusterDir, step, templateDir, extraArgs...)
}

func buildClusterClient(kubeconfig string) (*clientset.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to build config: %v", err)
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to build client: %v", err)
	}

	return client, nil
}
