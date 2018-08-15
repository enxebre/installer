package workflow

import (
	"fmt"

	"path/filepath"

	"github.com/openshift/installer/installer/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

const (
	kubeconfigPath = "generated/auth/kubeconfig"
)

// DestroyWorkflow creates new instances of the 'destroy' workflow,
// responsible for running the actions required to remove resources
// of an existing cluster and clean up any remaining artefacts.
func DestroyWorkflow(clusterDir string) Workflow {
	return Workflow{
		metadata: metadata{clusterDir: clusterDir},
		steps: []step{
			refreshConfigStep,
			destroyJoinWorkersStep,
			destroyJoinMastersStep,
			destroyEtcdStep,
			destroyBootstrapStep,
			destroyTNCDNSStep,
			destroyTopologyStep,
			destroyAssetsStep,
		},
	}
}

func destroyAssetsStep(m *metadata) error {
	return runDestroyStep(m, assetsStep)
}

func destroyEtcdStep(m *metadata) error {
	return runDestroyStep(m, etcdStep)
}

func destroyBootstrapStep(m *metadata) error {
	return runDestroyStep(m, mastersStep, []string{bootstrapOff}...)
}

func destroyTNCDNSStep(m *metadata) error {
	return destroyTNCDNS(m)
}

func destroyTopologyStep(m *metadata) error {
	return runDestroyStep(m, topologyStep)
}

func destroyJoinWorkersStep(m *metadata) error {
	if m.cluster.Platform == config.PlatformAWS {
		// we don't want to return error here as for cases
		// where the api is not available we want the destroy process to continue
		deleteWorkerMachineSet(filepath.Join(m.clusterDir, kubeconfigPath))
		return nil

	}
	return runDestroyStep(m, joinWorkersStep)
}

func deleteWorkerMachineSet(kubeconfig string) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed building kube config for machineset: %v", err)
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed building kube client for machineset: %v", err)
	}

	err = client.ClusterV1alpha1().MachineSets("test").Delete("worker", &v1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed deleting machineset: %v", err)
	}
	return nil
}

func destroyJoinMastersStep(m *metadata) error {
	return runDestroyStep(m, mastersStep, []string{bootstrapOff}...)
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
