package authz

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/vault-plugins/shared/client"
)

var period = time.Minute * 5

func RunVaultPoliciesGarbageCollector(vaultClientProvider *client.VaultClientController, logger hclog.Logger) {
	for {
		apiClient, err := (*vaultClientProvider).APIClient(nil)
		switch {
		case err != nil && !errors.Is(err, client.ErrNotInit):
			logger.Error(fmt.Sprintf("error getting apiClient:%s", err.Error()))
		case err == nil && apiClient == nil:
			logger.Error("error getting apiClient is nil, but got nil apiClient")
		case errors.Is(err, client.ErrNotInit):
			logger.Debug(fmt.Sprintf("getting apiClient:%s", err.Error()))
		}
		if err == nil && apiClient != nil {
			checkAndRemoveOverduePolicies(newVaultAPIClientBasedService(apiClient), logger)
		} else {
			time.Sleep(time.Second * 30)
		}
	}
}

type VaultPolicyService interface {
	ListVaultPolicies() ([]string, error) // returns policy_names
	DeleteVaultPolicies(policiesNames []string, logger hclog.Logger)
}

type vaultAPIClientBasedService struct {
	vaultAPIClient *api.Client
}

func (v *vaultAPIClientBasedService) ListVaultPolicies() ([]string, error) {
	return v.vaultAPIClient.Sys().ListPolicies()
}

func (v *vaultAPIClientBasedService) DeleteVaultPolicies(policiesNames []string, logger hclog.Logger) {
	errorCounter := 0
	counter := 0
	for _, p := range policiesNames {
		err := v.vaultAPIClient.Sys().DeletePolicy(p)
		if err != nil {
			errorCounter++
			logger.Error(fmt.Sprintf("deleting policy :'%s':%s", p, err.Error()))
		} else {
			counter++
		}
	}
	logger.Info(fmt.Sprintf("succseeded deleted %d overdue vault policies, got error for %d deletions", counter, errorCounter))
}

func newVaultAPIClientBasedService(vaultAPIClient *api.Client) VaultPolicyService {
	return &vaultAPIClientBasedService{vaultAPIClient}
}

func checkAndRemoveOverduePolicies(vaultPolicyService VaultPolicyService, logger hclog.Logger) {
	nextLoopStartTimeStamp := time.Now().Add(period)
	allPolicies, err := vaultPolicyService.ListVaultPolicies()
	if err != nil {
		logger.Error(fmt.Sprintf("collecting vault policies:%s", err.Error()))
	} else {
		overduePolicies := collectOverduePolicies(allPolicies, logger)
		deletePolicies(overduePolicies, vaultPolicyService, logger)
	}
	time.Sleep(time.Until(nextLoopStartTimeStamp))
}

func collectOverduePolicies(policies []string, logger hclog.Logger) []string {
	var overduePolicies []string
	for _, policy := range policies {
		overdue, err := IsVaultPolicyOverdue(policy)
		if err != nil {
			logger.Error(fmt.Sprintf("checking vault policy for overdue: %s", err.Error()))
		}
		if overdue {
			overduePolicies = append(overduePolicies, policy)
		}
	}
	return overduePolicies
}

const bunch = 50

// deletePolicies delete policies with pauses to not overload  vault
func deletePolicies(policies []string, service VaultPolicyService, logger hclog.Logger) {
	if len(policies) == 0 {
		logger.Info("no policies for deleting")
		return
	}
	if len(policies) < bunch {
		service.DeleteVaultPolicies(policies, logger)
		return
	}
	startTimeStamp := time.Now()
	service.DeleteVaultPolicies(policies[0:bunch], logger)
	workInterval := time.Since(startTimeStamp)
	amountOfBunches := int64(math.Ceil(float64(len(policies)) / bunch))
	allPausedurationNS := (period.Nanoseconds() - workInterval.Nanoseconds()*amountOfBunches) / 2
	if allPausedurationNS <= 0 {
		service.DeleteVaultPolicies(policies[0:bunch], logger)
		return
	}
	pause := time.Duration(allPausedurationNS / amountOfBunches)
	for i := 1; i < int(amountOfBunches); i++ {
		time.Sleep(pause)
		startBunch := i * bunch
		endBunch := (i + 1) * bunch
		if endBunch > len(policies) {
			endBunch = len(policies)
		}
		service.DeleteVaultPolicies(policies[startBunch:endBunch], logger)
	}
}
