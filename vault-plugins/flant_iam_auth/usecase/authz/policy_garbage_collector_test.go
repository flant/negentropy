package authz

import (
	"fmt"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func Test_collectOverduePolicies(t *testing.T) {
	mockLog := hclog.NewNullLogger()
	validTill := time.Now().Add(time.Minute)
	validPolicies := generatePoliciesNames(validTill, 5)
	overdueValidTill := time.Now().Add(-time.Minute)
	overduePolicies := generatePoliciesNames(overdueValidTill, 5)
	policies := append(validPolicies, overduePolicies...) // nolint: gocritic

	actualOverduePolicies := collectOverduePolicies(policies, mockLog)

	require.Equal(t, overduePolicies, actualOverduePolicies)
}

func generatePoliciesNames(validTill time.Time, n int) []string {
	overduePolicies := []string{}
	for i := 0; i < n; i++ {
		policy := VaultPolicy{Name: uuid.New()}
		policy.AddValidTillToName(validTill)
		overduePolicies = append(overduePolicies, policy.Name)
	}
	return overduePolicies
}

func Test_deletePoliciesOneBunch(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish() // it will mark this example test as failed because there are no calls to formatterMock.Format() and readCloserMock.Read() below
	mockService := NewVaultPolicyServiceMock(t)
	mockLog := hclog.NewNullLogger()
	policies := []string{"p1", "p2"}
	mockService.DeleteVaultPoliciesMock.Expect(policies, mockLog)

	deletePolicies(policies, mockService, mockLog)
}

func Test_deletePoliciesTwoBunches(t *testing.T) {
	// It should be to calls with pause
	mc := minimock.NewController(t)
	defer mc.Finish() // it will mark this example test as failed because there are no calls to formatterMock.Format() and readCloserMock.Read() below
	start := time.Now()
	mockService := NewVaultPolicyServiceMock(t)
	mockLog := hclog.NewNullLogger()
	expectedPolicies := make([]string, 0, 75)
	for i := 1; i < 75; i++ {
		expectedPolicies = append(expectedPolicies, fmt.Sprintf("%d", i))
	}
	period = time.Millisecond * 500
	counter := 0
	mockService.DeleteVaultPoliciesMock.Set(func(policiesNames []string, logger hclog.Logger) {
		startBunch := counter * bunch
		endBunch := (counter + 1) * bunch
		if endBunch > len(expectedPolicies) {
			endBunch = len(expectedPolicies)
		}
		counter++
		require.Equal(t, expectedPolicies[startBunch:endBunch], policiesNames)
		time.Sleep(time.Millisecond * 5)
	})

	deletePolicies(expectedPolicies, mockService, mockLog)

	require.Greater(t, time.Since(start).Nanoseconds(), (time.Millisecond * 125).Nanoseconds())
	require.Less(t, time.Since(start).Nanoseconds(), (time.Millisecond * 250).Nanoseconds())
	require.Equal(t, 2, counter)
}
