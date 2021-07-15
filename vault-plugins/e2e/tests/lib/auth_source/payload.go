package auth_source

import (
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

type SourceForTest struct {
	Source         *model.AuthSource
	Name           string
	ExpectedEaName func(object io.MemoryStorableObject) string
}

func (s *SourceForTest) ToPayload() (map[string]interface{}, string) {
	mp := tools.ToMap(s.Source)
	delete(mp, "uuid")
	delete(mp, "name")

	return mp, s.Source.Name
}

var (
	JWTWithEaNameEmail = SourceForTest{
		Name: "email",
		Source: &model.AuthSource{
			UUID: utils.UUID(),
			Name: "s1",

			JWTValidationPubKeys: []string{JWTPubKey},
			JWTSupportedAlgs:     []string{},
			OIDCResponseTypes:    []string{},
			BoundIssuer:          "http://vault.example.com/",
			NamespaceInState:     true,
			EntityAliasName:      model.EntityAliasNameEmail,
		},

		ExpectedEaName: func(object io.MemoryStorableObject) string {
			if object.ObjType() == iam.UserType {
				return object.(*iam.User).Email
			}

			return ""
		},
	}

	JWTWithEaNameFullID = SourceForTest{
		Name: "full_id",
		Source: &model.AuthSource{
			UUID: utils.UUID(),
			Name: "s2",

			JWTValidationPubKeys: []string{JWTPubKey},
			JWTSupportedAlgs:     []string{},
			OIDCResponseTypes:    []string{},
			BoundIssuer:          "http://vault.example.com/",
			NamespaceInState:     true,
			EntityAliasName:      model.EntityAliasNameFullIdentifier,
		},

		ExpectedEaName: func(object io.MemoryStorableObject) string {
			if object.ObjType() == iam.UserType {
				return object.(*iam.User).FullIdentifier
			}

			return ""
		},
	}

	JWTWithEaNameUUID = SourceForTest{
		Name: "uuid",
		Source: &model.AuthSource{
			UUID: utils.UUID(),
			Name: "s3",

			JWTValidationPubKeys: []string{JWTPubKey},
			JWTSupportedAlgs:     []string{},
			OIDCResponseTypes:    []string{},
			BoundIssuer:          "http://vault.example.com/",
			NamespaceInState:     true,
			EntityAliasName:      model.EntityAliasNameUUID,
		},

		ExpectedEaName: func(object io.MemoryStorableObject) string {
			if object.ObjType() == iam.UserType {
				return object.(*iam.User).UUID
			}
			return ""
		},
	}

	JWTWithEaNameUUIDEnableSa = SourceForTest{
		Name: "enable sa uuid",
		Source: &model.AuthSource{
			UUID: utils.UUID(),
			Name: "s4",

			JWTValidationPubKeys: []string{JWTPubKey},
			JWTSupportedAlgs:     []string{},
			OIDCResponseTypes:    []string{},
			BoundIssuer:          "http://vault.example.com/",
			NamespaceInState:     true,
			AllowServiceAccounts: true,
			EntityAliasName:      model.EntityAliasNameUUID,
		},

		ExpectedEaName: func(object io.MemoryStorableObject) string {
			switch object.ObjType() {
			case iam.UserType:
				return object.(*iam.User).UUID
			case iam.ServiceAccountType:
				return object.(*iam.ServiceAccount).UUID
			}

			return ""
		},
	}

	JWTWithEaNameFullIDEnableSa = SourceForTest{
		Name: "enable sa full_id",
		Source: &model.AuthSource{
			UUID: utils.UUID(),
			Name: "s5",

			JWTValidationPubKeys: []string{JWTPubKey},
			JWTSupportedAlgs:     []string{},
			OIDCResponseTypes:    []string{},
			BoundIssuer:          "http://vault.example.com/",
			NamespaceInState:     true,
			AllowServiceAccounts: true,
			EntityAliasName:      model.EntityAliasNameFullIdentifier,
		},

		ExpectedEaName: func(object io.MemoryStorableObject) string {
			switch object.ObjType() {
			case iam.UserType:
				return object.(*iam.User).FullIdentifier
			case iam.ServiceAccountType:
				return object.(*iam.ServiceAccount).FullIdentifier
			}

			return ""
		},
	}

	JWTWithEaNameEmailEnableSa = SourceForTest{
		Name: "enable sa email",
		Source: &model.AuthSource{
			UUID: utils.UUID(),
			Name: "s6",

			JWTValidationPubKeys: []string{JWTPubKey},
			JWTSupportedAlgs:     []string{},
			OIDCResponseTypes:    []string{},
			BoundIssuer:          "http://vault.example.com/",
			NamespaceInState:     true,
			AllowServiceAccounts: true,
			EntityAliasName:      model.EntityAliasNameEmail,
		},

		ExpectedEaName: func(object io.MemoryStorableObject) string {
			if object.ObjType() == iam.UserType {
				return object.(*iam.User).Email
			}
			return ""
		},
	}

)

func GenerateSources() []SourceForTest {
	return []SourceForTest{
		JWTWithEaNameEmail,
		JWTWithEaNameFullID,
		JWTWithEaNameUUID,
		JWTWithEaNameUUIDEnableSa,
		JWTWithEaNameFullIDEnableSa,
		// JWTWithEaNameEmailEnableSa,
	}
}