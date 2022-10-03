package kube

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/logical"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubeService interface {
	RunJob(ctx context.Context, hashCommit string, vaultsB64Json string) error
	IsJobFinished(ctx context.Context, hashCommit string) (bool, error)
}

var StorageKeyConfiguration = "k8s_configuration"

func NewKubeService(ctx context.Context, storage logical.Storage) (KubeService, error) {
	entry, err := storage.Get(ctx, StorageKeyConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to adress storage: %w", err)
	}
	var kconfig *rest.Config
	if entry != nil {
		err := json.Unmarshal(entry.Value, kconfig)
		if err != nil {
			return nil, fmt.Errorf("failed parsing kube config from storage: %w", err)
		}
	} else {
		kconfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(kconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s clientset: %w", err)
	}

	return &kubeService{
		kubeNameSpace: "negentropy-dev",
		clientset:     clientset,
	}, nil
}

type kubeService struct {
	kubeNameSpace string
	clientset     *kubernetes.Clientset
}

//go:embed job_template.yaml
var jobTemplate string

func (k *kubeService) RunJob(ctx context.Context, hashCommit string, vaultsB64Json string) error {
	specStr := strings.Replace(jobTemplate, "COMMIT_PLACEHOLDER", hashCommit, 1)
	specStr = strings.Replace(jobTemplate, "VAULTS_B64_PLACEHOLDER", vaultsB64Json, 1)

	var spec batchv1.Job
	err := yaml.Unmarshal([]byte(specStr), &spec)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}
	spec.ObjectMeta.Name = hashCommit
	spec.ObjectMeta.Namespace = k.kubeNameSpace
	jobs := k.clientset.BatchV1().Jobs(k.kubeNameSpace)
	_, err = jobs.Create(ctx, &spec, metav1.CreateOptions{})
	return err
}

func (k *kubeService) IsJobFinished(ctx context.Context, hashCommit string) (bool, error) {
	jobs := k.clientset.BatchV1().Jobs(k.kubeNameSpace)
	job, err := jobs.Get(ctx, hashCommit, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("obtaining data: %w", err)
	}
	if job.Status.Failed > 0 || job.Status.Succeeded > 0 {
		return true, nil
	}
	for _, c := range job.Status.Conditions {
		if (c.Type == "Complete" || c.Type == "Failed") &&
			c.Status == "True" {
			return true, nil
		}
	}
	return false, nil
}
