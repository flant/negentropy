package internal

import (
	"encoding/json"
	"fmt"
	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/api"
	"net/http"
	"strings"
)

const AuthPluginSelfTopic = "authplugin_self_topic"
const AuthPluginRootReplicaTopic = "authplugin_root_replica_topic"
const IamPluginSelfTopic = "iamplugin_self_topic"
const JwksTopic = "jwks_topic"
const MultipassNumTopic = "multipass_nem_topic"

func CleanTopic(summary *SummaryOfTopic) ([]string, error) {
	switch summary.Topic.Type {
	case AuthPluginSelfTopic:
		return cleanAuthSelfTopic(summary)
	default:
		return nil, fmt.Errorf("topic with type %q is not cleanable yet", summary.Topic.Type)
	}
	return nil, nil
}

func cleanAuthSelfTopic(summary *SummaryOfTopic) ([]string, error) {
	originVault := summary.Topic.OriginVault
	if originVault == nil || originVault.RootToken == "" || originVault.Url == "" {
		return nil, fmt.Errorf("empty origin vault: %#v", *originVault)
	}
	vaultClient, err := clientWithToken(*originVault)
	if err != nil {
		return nil, err
	}
	resultErr := multierror.Error{}
	result := []string{}
	itemTypes := []Type{"auth_method", "auth_source", "policy"}         // what itemTypes should be deleted
	erasers := []func(vaultClient *api.Client, item ItemSummary) error{ // should suit itemTypes
		eraseAuthMethod, eraseAuthSource, erasePolicy,
	}
	for i, eraser := range erasers {
		itemType := itemTypes[i]
		count, err := cleanItems(vaultClient, summary.Summaries[itemType], eraser)
		if err != nil {
			resultErr.Errors = append(resultErr.Errors, err)
		}
		result = append(result, fmt.Sprintf("%s : deleted: %d", itemType, count))

	}
	return result, resultErr.ErrorOrNil()
}

func cleanItems(client *api.Client, items map[ItemKey]ItemSummary, itemEraser func(vaultClient *api.Client, item ItemSummary) error) (int, error) {
	count := 0
	for _, summary := range items {
		if !summary.IsDeleted {
			err := itemEraser(client, summary)
			if err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

func eraseAuthMethod(vaultClient *api.Client, methodSummary ItemSummary) error {
	_, err := vaultClient.Logical().Delete(lib.IamAuthPluginPath + "/" + methodSummary.Key)
	return err
}

func eraseAuthSource(vaultClient *api.Client, sourceSummary ItemSummary) error {
	_, err := vaultClient.Logical().Delete(lib.IamAuthPluginPath + "/" + sourceSummary.Key)
	return err
}

func clientWithToken(vault Vault) (*api.Client, error) {
	cfg := api.DefaultConfig()
	transport := cfg.HttpClient.Transport.(*http.Transport)
	transport.TLSClientConfig.InsecureSkipVerify = true
	cl, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	cl.SetToken(vault.RootToken)
	err = cl.SetAddress(vault.Url)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

func erasePolicy(vaultClient *api.Client, policySummary ItemSummary) error {
	splitted := strings.Split(policySummary.Key, "/")
	policyName := splitted[1]

	resp, err := vaultClient.Logical().Read("auth/flant/login_policy/" + policyName)
	if err != nil {
		return err
	}
	policyRaw := resp.Data["policy"].(map[string]interface{})
	if policyRaw["archiving_timestamp"] == json.Number("0") {
		resp, err = vaultClient.Logical().Delete("auth/flant/login_policy/" + policyName)
		if err != nil {
			return err
		}
	}
	_, err = vaultClient.Logical().Delete("auth/flant/login_policy/" + policyName + "/erase")
	return err
}
