package fixtures

import (
	"encoding/hex"
	"math/rand"
	"time"

	iam_fixtures "github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	PolicyName1 = "policy1"
	PolicyName2 = "policy2"
)

func Policies() []model.Policy {
	return []model.Policy{
		{
			ArchiveMark: memdb.ArchiveMark{},
			Name:        PolicyName1,
			Rego:        "regogo1", // TODO
			Roles:       []string{iam_fixtures.RoleName1, iam_fixtures.RoleName2},
			ClaimSchema: "claim_schema1", // TODO
		},
		{
			ArchiveMark: memdb.ArchiveMark{},
			Name:        PolicyName2,
			Rego:        "regogo2", // TODO
			Roles:       []string{iam_fixtures.RoleName1, iam_fixtures.RoleName2},
			ClaimSchema: "claim_schema2", // TODO
		},
	}
}

func RandomPolicyCreatePayload() map[string]interface{} {
	policySet := Policies()
	rand.Seed(time.Now().UnixNano())
	sample := policySet[rand.Intn(len(policySet))]
	return map[string]interface{}{
		"name":         "policy_" + RandomStr(),
		"rego":         sample.Rego,
		"roles":        sample.Roles,
		"claim_schema": sample.ClaimSchema,
	}
}

func RandomStr() string {
	rand.Seed(time.Now().UnixNano())

	entityName := make([]byte, 20)
	_, err := rand.Read(entityName)
	if err != nil {
		panic("not generate entity name")
	}

	return hex.EncodeToString(entityName)
}
