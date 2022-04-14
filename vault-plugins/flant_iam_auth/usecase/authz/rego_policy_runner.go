package authz

import (
	"context"
	"encoding/json"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"

	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
)

type RegoPolicy = string

type UserData struct{}

type LoginClaims = map[string]interface{}

type RegoResult struct {
	Allow             bool
	BestEffectiveRole *iam_usecase.EffectiveRole
	VaultRules        []Rule
	TTL               string
	MaxTTL            string
}

type rawRegoResult struct {
	Allow         bool                        `json:"allow"`
	FilteredRoles []iam_usecase.EffectiveRole `json:"filtered_bindings"`
	VaultRules    []Rule                      `json:"rules"`
	TTL           string                      `json:"ttl"`
	MaxTTL        string                      `json:"max_ttl"`
}

// ApplyRegoPolicy parse all arguments and run rego policy
// parse result of rego policy run, and choose the best role_binding
func ApplyRegoPolicy(ctx context.Context, regoPolicy RegoPolicy, userData UserData,
	effectiveRoles []iam_usecase.EffectiveRole, claims LoginClaims) (*RegoResult, error) {
	data := map[string]interface{}{"effective_roles": effectiveRoles, "user_data": userData}
	store := inmem.NewFromObject(data)
	rego := rego.New(
		rego.Store(store),
		rego.Query("data.negentropy"),
		rego.Module("negentropy.rego", regoPolicy),
		rego.Input(claims),
	)

	// Run evaluation.
	rs, err := rego.Eval(ctx)
	if err != nil {
		return nil, err
	}
	tmp := rs[0].Expressions[0].Value
	d, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	var rawResult rawRegoResult
	err = json.Unmarshal(d, &rawResult)
	if err != nil {
		return nil, err
	}
	result := RegoResult{
		Allow: rawResult.Allow,
	}
	if !result.Allow {
		return &result, nil
	}
	result.VaultRules = rawResult.VaultRules
	result.TTL = rawResult.TTL
	result.MaxTTL = rawResult.MaxTTL
	bestRole, goodRole, someRole := rangeRoles(rawResult.FilteredRoles)
	switch {
	case bestRole != nil:
		result.BestEffectiveRole = bestRole
	case goodRole != nil:
		result.BestEffectiveRole = goodRole
	case someRole != nil:
		result.BestEffectiveRole = someRole
	}

	return &result, nil
}

func rangeRoles(rs []iam_usecase.EffectiveRole) (*iam_usecase.EffectiveRole, *iam_usecase.EffectiveRole, *iam_usecase.EffectiveRole) {
	var goodRolebinding, someRolebinding *iam_usecase.EffectiveRole
	for _, r := range rs {
		role := r
		if !r.RequireMFA && r.NeedApprovals == 0 {
			return &role, nil, nil
		}
		if r.RequireMFA && r.NeedApprovals == 0 && goodRolebinding == nil {
			goodRolebinding = &role
		}
		if r.NeedApprovals > 0 && (someRolebinding == nil || someRolebinding.NeedApprovals > r.NeedApprovals) {
			someRolebinding = &role
		}
	}
	return nil, goodRolebinding, someRolebinding
}
