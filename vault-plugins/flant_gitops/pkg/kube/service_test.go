package kube

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

func Test_Create(t *testing.T) {
	config := &restclient.Config{
		// TODO: switch to using cluster DNS.
		Host:            "https://api.negentropy.flant.dev",
		TLSClientConfig: restclient.TLSClientConfig{},
		BearerToken:     "eyJhbGciOiJSUzI1NiIsImtpZCI6ImQ2aXpWU2RQRlg3SkR2UjFjdk4xcVJhY2puQmVQRnZyMDNJTkZuUnFVUjAifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJuZWdlbnRyb3B5LWRldiIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJnaXRvcHMtdG9rZW4tMjY0ZzUiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoiZ2l0b3BzIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQudWlkIjoiNzZiNWI3NzktODY5Ni00ZjUyLThkNWQtNzcyMDI0NmZhMmZkIiwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Om5lZ2VudHJvcHktZGV2OmdpdG9wcyJ9.Tr6u8pXTKo59_D67BtvyYIUoias_U25XB-Jiju24d99nW4wjM55PDTmAoFD1PR3oxaW-AvH1nNY1ZV19psYsYe1KszdScEiUB0Burt3gscnnIGd3mYOoG0BAvQ3Fuq0iPgg5rGdOeSpavly_cpD7mSx5T20DGNtj1hxZiD7fj0fVc8IrvtxQPxV2hj9H7aw2f383nQIfBLVDnXkzExweo4QVXyj69cir5BUIoKUEGhs5nZLscJiAbWFXeABynHiQweQPthmxaXNMrmHMPGrHlnpZ8WogFu154EwW6SBylE4hxMoI1pwpJZIV0IcFssdQZCtQ47d7XX9EhSAoWIc8Lw",
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		println(err.Error())
		t.Fatalf("Failed to create K8s clientset")
	}

	service := kubeService{
		kubeNameSpace: "negentropy-dev",
		clientset:     clientset,
	}

	ctx := context.Background()
	err = service.RunJob(ctx, "la-la-la", "bbbbb")
	if err != nil {
		println(err.Error())
		t.Fatalf("Failed to job")
	}

	finished := false
	for !finished {
		finished, err = service.IsJobFinished(ctx, "la-la-la")
		if err != nil {
			println(err.Error())
			t.Fatalf("Failed to check is finished")
		}
		time.Sleep(time.Second)
	}

}

func Test_NewService(t *testing.T) {
	storage := &logical.InmemStorage{}
	_, err := NewKubeService(context.Background(), storage)
	if err != nil {
		println(err.Error())
		t.Fatalf("Failed to job")
	}
}
