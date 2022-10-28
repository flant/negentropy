package kube

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type (
	exist    = bool
	finished = bool
)

type KubeService interface {
	RunJob(ctx context.Context, hashCommit string, vaultsB64Json string, logger log.Logger) error
	CheckJob(ctx context.Context, hashCommit string) (exist, finished, error)
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
	const nameSpaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	data, err := ioutil.ReadFile(nameSpaceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read pod namespace file: %w", err)
	}

	namespace := string(data)

	return &kubeService{
		kubeNameSpace: namespace,
		clientset:     clientset,
	}, nil
}

type kubeService struct {
	kubeNameSpace string
	clientset     *kubernetes.Clientset
}

//go:embed job_template.yaml
var jobTemplate string

func (k *kubeService) RunJob(ctx context.Context, hashCommit string, vaultsB64Json string, logger log.Logger) error {
	specStr := replacePlaceholders(jobTemplate, hashCommit, vaultsB64Json)
	logger.Debug("replacePlaceholders", "spec", specStr)
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

func (k *kubeService) CheckJob(ctx context.Context, hashCommit string) (exist, finished, error) {
	jobs := k.clientset.BatchV1().Jobs(k.kubeNameSpace)
	job, err := jobs.Get(ctx, hashCommit, metav1.GetOptions{})
	if notFoundErr(err, hashCommit) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("obtaining data: %w", err)
	}
	if job.Status.Failed > 0 || job.Status.Succeeded > 0 {
		return true, true, nil
	}
	for _, c := range job.Status.Conditions {
		if (c.Type == "Complete" || c.Type == "Failed") &&
			c.Status == "True" {
			return true, true, nil
		}
	}
	return true, false, nil
}

// check is error : `jobs.batch "JOB_NAME" not found`
func notFoundErr(err error, jobName string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.HasPrefix(msg, "jobs.batch") && strings.HasSuffix(msg, "not found") && strings.Contains(msg, jobName)
}

func replacePlaceholders(template string, hashCommit string, vaultsB64Json string) string {
	specStr := strings.ReplaceAll(template, "COMMIT_PLACEHOLDER", hashCommit)
	specStr = strings.ReplaceAll(specStr, "VAULTS_B64_PLACEHOLDER", vaultsB64Json)
	return specStr
}
