package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/e2e/tests/lib"
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
	case IamPluginSelfTopic:
		return cleanIamSelfTopic(summary)
	default:
		return nil, fmt.Errorf("topic with type %q is not cleanable", summary.Topic.Type)
	}
}

func cleanAuthSelfTopic(summary *SummaryOfTopic) ([]string, error) {
	itemTypes := []Type{"auth_method", "auth_source", "policy"}         // what itemTypes should be deleted
	erasers := []func(vaultClient *api.Client, item ItemSummary) error{ // should suit itemTypes
		eraseAuthMethod, eraseAuthSource, erasePolicy,
	}
	return cleanTopic(summary, itemTypes, erasers)
}

func cleanTopic(summary *SummaryOfTopic, itemTypes []Type, erasers []func(vaultClient *api.Client, item ItemSummary) error) ([]string, error) {
	vaultClient, err := clientWithToken(summary)
	if err != nil {
		return nil, err
	}
	resultErr := multierror.Error{}
	result := []string{}
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
	for {
		nextItems := map[ItemKey]ItemSummary{}
		if len(items) == 0 {
			break
		}
		for k, summary := range items {
			if summary.IsDeleted {
				continue
			}
			err := itemEraser(client, summary)
			if errors.Is(err, errRepeatLater) {
				nextItems[k] = summary
				continue
			}
			if err != nil {
				panic(err)
			}
			count++
		}
		items = nextItems
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

func clientWithToken(summary *SummaryOfTopic) (*api.Client, error) {
	originVault := summary.Topic.OriginVault
	if originVault == nil || originVault.RootToken == "" || originVault.Url == "" {
		return nil, fmt.Errorf("empty origin vault: %#v", *originVault)
	}

	cfg := api.DefaultConfig()
	transport := cfg.HttpClient.Transport.(*http.Transport)
	transport.TLSClientConfig.InsecureSkipVerify = true
	cl, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	cl.SetToken(originVault.RootToken)
	err = cl.SetAddress(originVault.Url)
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

// cleanIamSelfTopic
func cleanIamSelfTopic(summary *SummaryOfTopic) ([]string, error) {
	itemTypes := []Type{"tenant", "teammate"}                           // what itemTypes should be deleted
	erasers := []func(vaultClient *api.Client, item ItemSummary) error{ // should suit itemTypes
		eraseWholeTenant, eraseWholeTeam,
	}
	return cleanTopic(summary, itemTypes, erasers)
}

func eraseWholeTenant(vaultClient *api.Client, tenantSummary ItemSummary) error {
	tenantKey := tenantSummary.Key // tenant/{tenant_uuid}
	basePath := "flant/" + tenantKey
	resp, err := vaultClient.Logical().Read(basePath)
	if err != nil {
		return err
	}
	tenantRaw := resp.Data["tenant"].(map[string]interface{})
	if tenantRaw["origin"] == "flant_flow_predefined" {
		// skip, can't delete
		return nil
	}
	if tenantRaw["origin"] == "flant_flow" {
		splitted := strings.Split(tenantKey, "/")
		basePath = "flant/client/" + splitted[1]
	}
	if tenantRaw["archiving_timestamp"] == json.Number("0") {
		resp, err = vaultClient.Logical().Delete(basePath)
		if err != nil {
			return err
		}
	}
	_, err = vaultClient.Logical().Delete(basePath + "/erase")
	return err
}

var errRepeatLater = fmt.Errorf("object is linked to some object the same type, repeat later")

func eraseWholeTeam(vaultClient *api.Client, teamSummary ItemSummary) error {
	err := checkCanBeDeleted(vaultClient, teamSummary.Key)
	if err != nil {
		return err
	}

	request := vaultClient.NewRequest("GET", "/v1/flant/"+teamSummary.Key+"/teammate")
	request.URL.Path = request.URL.Path + "/"
	request.Params = url.Values{"show_archived": {"true"}}

	resp, err := vaultClient.RawRequest(request)
	if err != nil {
		return err
	}
	secret, err := api.ParseSecret(resp.Body)
	teammatesRaw := secret.Data["teammates"].([]interface{})
	for _, teammateRaw := range teammatesRaw {
		teammate := teammateRaw.(map[string]interface{})
		teammateUUID := teammate["uuid"].(string)
		basePath := "flant/" + teamSummary.Key + "/teammate/" + teammateUUID
		if teammate["archiving_timestamp"] == json.Number("0") {
			_, err = vaultClient.Logical().Delete(basePath)
			if err != nil {
				return err
			}
		}
		_, err = vaultClient.Logical().Delete(basePath + "/erase")
		if err != nil {
			return err
		}
	}

	secret, err = vaultClient.Logical().Read("flant/" + teamSummary.Key)
	if err != nil {
		return err
	}
	teamRaw := secret.Data["team"].(map[string]interface{})
	if teamRaw["archiving_timestamp"] == json.Number("0") {
		_, err = vaultClient.Logical().Delete("flant/" + teamSummary.Key)
		if err != nil {
			return err
		}
	}
	_, err = vaultClient.Logical().Delete("flant/" + teamSummary.Key + "/erase")
	return err
}

func checkCanBeDeleted(vaultClient *api.Client, teamKafkaKey string) error {
	teamUUID := extractID(teamKafkaKey)
	println(teamUUID)
	request := vaultClient.NewRequest("GET", "/v1/flant/team")
	request.URL.Path = request.URL.Path + "/"
	request.Params = url.Values{"show_archived": {"true"}}

	resp, err := vaultClient.RawRequest(request)
	if err != nil {
		return err
	}
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return err
	}
	found := false // TODO REMOVE
	teamsRaw := secret.Data["teams"].([]interface{})
	for _, teamRaw := range teamsRaw {
		checkTeam := teamRaw.(map[string]interface{})
		checkTeamIdentifier := checkTeam["identifier"].(string)
		checkTeamUUID := checkTeam["uuid"].(string)
		checkParentTeam := checkTeam["parent_team_uuid"].(string)
		if checkParentTeam == teamUUID {
			return fmt.Errorf("%w: %q is parent for %q team", errRepeatLater, teamUUID, checkTeamIdentifier)
		}
		if teamUUID == checkTeamUUID { // TODO REMOVE
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%q team is erased", teamUUID)
	}
	return nil

}

func extractID(kafkeKey string) string {
	splitted := strings.Split(kafkeKey, "/")
	if len(splitted) != 2 {
		panic(fmt.Sprintf("wrong key to extract ID: %s", kafkeKey))
	}
	return splitted[1]
}
