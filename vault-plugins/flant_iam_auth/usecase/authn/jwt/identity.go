package jwt

import (
	"fmt"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"golang.org/x/oauth2"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
)

// CreateIdentity creates an alias and set of groups aliases based on the authMethodConfig
// definition and received claims.
func CreateIdentity(logger log.Logger, allClaims map[string]interface{}, authMethod *model.AuthMethod, _ oauth2.TokenSource) (*logical.Alias, []string, error) {
	userClaimRaw, ok := allClaims[authMethod.UserClaim]
	if !ok {
		return nil, nil, fmt.Errorf("claim %q not found in token", authMethod.UserClaim)
	}
	userName, ok := userClaimRaw.(string)
	if !ok {
		return nil, nil, fmt.Errorf("claim %q could not be converted to string", authMethod.UserClaim)
	}

	metadata, err := extractMetadata(logger, allClaims, authMethod.ClaimMappings)
	if err != nil {
		return nil, nil, err
	}

	alias := &logical.Alias{
		Name:     userName,
		Metadata: metadata,
	}

	var groupAliases []string

	if authMethod.GroupsClaim == "" {
		return alias, groupAliases, nil
	}

	groupsClaimRaw := GetClaim(logger, allClaims, authMethod.GroupsClaim)

	if groupsClaimRaw == nil {
		return nil, nil, fmt.Errorf("%q claim not found in token", authMethod.GroupsClaim)
	}

	groups, ok := NormalizeList(groupsClaimRaw)

	if !ok {
		return nil, nil, fmt.Errorf("%q claim could not be converted to string list", authMethod.GroupsClaim)
	}
	for _, groupRaw := range groups {
		group, ok := groupRaw.(string)
		if !ok {
			return nil, nil, fmt.Errorf("value %v in groups claim could not be parsed as string", groupRaw)
		}
		if group == "" {
			continue
		}
		groupAliases = append(groupAliases, group)
	}

	return alias, groupAliases, nil
}
