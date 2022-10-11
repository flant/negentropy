package kube

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ReplacePlaceholders(t *testing.T) {
	hashcommit := "7f403b65ef40054d8782ae8fe0ba82a11c7fd9ca"
	vaultsB64json := "J1t7Im5hbWUiOiJ2YXVsdC1yb290LTEiLCAidXJsIjoiaHR0cDovLzEyNy4wLjAuMTo4MjAwLyIsICJ0b2tlbiI6Imh2cy5LcU42VFNDeXg5Q0ZiVHhDVkNPUndBU04ifV0n"

	result := replacePlaceholders(jobTemplate, hashcommit, vaultsB64json)

	require.True(t, strings.Contains(result, hashcommit))
	require.False(t, strings.Contains(result, "COMMIT_PLACEHOLDER"))
	require.True(t, strings.Contains(result, vaultsB64json))
	require.False(t, strings.Contains(result, "VAULTS_B64_PLACEHOLDER"))
}
