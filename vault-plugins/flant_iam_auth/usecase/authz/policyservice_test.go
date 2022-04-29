package authz

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Rule(t *testing.T) {
	r := &Rule{
		Path:               "/tenant/199908",
		Create:             true,
		Update:             true,
		Read:               true,
		Delete:             true,
		List:               true,
		AllowedParameters:  map[string][]string{"ap1": {"ap1_value1", "ap1_value2"}, "ap2": {"ap2_value1", "ap2_value2"}, "*": {}},
		RequiredParameters: []string{"ap1", "ap2"},
	}

	s := r.String()
	// println(s)

	require.Equal(t, "path \"/tenant/199908\" {\n   capabilities = [\"create\", \"update\", \"read\", \"delete\", \"list\"]\n   required_parameters = [\"ap1\", \"ap2\"]\n   allowed_parameters = {\n      \"ap1\" = [\"ap1_value1\", \"ap1_value2\"]\n      \"ap2\" = [\"ap2_value1\", \"ap2_value2\"]\n      \"*\" = []\n   }\n}", s)
}

func Test_Unmarshall(t *testing.T) {
	s := `[
       {
           "allowed_parameters": {
               "valid_principals": [
                   "2db561b02578945905f9688c540bc7489cf9dc7578d20b08cda636682c636a56",
                   "d56b1dfc8e81b509b007d0465f291524ccd4a5fb99f15eda5ecb6b57c47ba793"
               ]
           },
           "capabilities": [
               "update"
           ],
           "path": "ssh/sign/signer",
           "required_parameters": [
               "valid_principals"
           ]
       }
   ]`
	var rules []Rule
	err := json.Unmarshal([]byte(s), &rules)
	require.NoError(t, err)
	fmt.Printf("%#v", rules)
	expected := []Rule{{Path: "ssh/sign/signer", AllowedParameters: map[string][]string{"valid_principals": {
		"2db561b02578945905f9688c540bc7489cf9dc7578d20b08cda636682c636a56",
		"d56b1dfc8e81b509b007d0465f291524ccd4a5fb99f15eda5ecb6b57c47ba793",
	}}, RequiredParameters: []string{"valid_principals"}, Create: false, Update: true, Read: false, Delete: false, List: false}}
	require.Equal(t, expected, rules)
}
