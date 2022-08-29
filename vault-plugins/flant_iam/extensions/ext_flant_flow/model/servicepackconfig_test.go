package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TeammateList(t *testing.T) {
	servicePacks := map[ServicePackName]ServicePackCFG{
		L1: nil,
		DevOps: DevopsServicePackCFG{
			Team: "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
		},
		Mk8s:       nil,
		Deckhouse:  nil,
		Okmeter:    nil,
		Consulting: nil,
	}
	data, err := json.Marshal(servicePacks)
	require.NoError(t, err)
	var rawRestoredServicePack interface{}
	err = json.Unmarshal(data, &rawRestoredServicePack)
	require.NoError(t, err)

	restoredServicePack, ok := rawRestoredServicePack.(map[ServicePackName]interface{})
	require.Equal(t, true, ok)
	parsedServicePack, err := ParseServicePacks(restoredServicePack)
	require.NoError(t, err)
	require.Equal(t, servicePacks, parsedServicePack)
}
