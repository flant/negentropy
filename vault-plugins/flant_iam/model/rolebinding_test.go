package model

import (
	"fmt"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/sethvargo/go-password/password"
	"github.com/stretchr/testify/assert"
)

func Test_RoleBindingDbSchema(t *testing.T) {
	schema := RoleBindingSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("role binding schema is invalid: %v", err)
	}
}

type Model interface {
	ObjType() string
	ObjId() string
}

func toSubjectNotation(m Model) SubjectNotation {
	return SubjectNotation{
		Type: m.ObjType(),
		ID:   m.ObjId(),
	}
}

func toSubjectNotations(ms ...Model) []SubjectNotation {
	sns := make([]SubjectNotation, 0)
	for _, m := range ms {
		sns = append(sns, toSubjectNotation(m))
	}
	return sns
}

func Test_RoleBindingOnCreationSubjectsCalculation(t *testing.T) {
	store, _ := initTestDB()
	tx := store.Txn(true)

	// Data
	ten := genTenant(tx)
	sa1 := genServiceAccount(tx, ten.UUID)
	sa2 := genServiceAccount(tx, ten.UUID)
	u1 := genUser(tx, ten.UUID)
	u2 := genUser(tx, ten.UUID)
	g1 := genGroup(tx, ten.UUID)
	g2 := genGroup(tx, ten.UUID)

	tests := []struct {
		name       string
		subjects   []Model
		assertions func(*testing.T, *RoleBinding)
	}{
		{
			name:     "no subjects",
			subjects: []Model{},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 0, "should contain no serviceaccounts")
				assert.Len(t, rb.Users, 0, "should contain no users")
				assert.Len(t, rb.Groups, 0, "should contain no groups")
			},
		},
		{
			name:     "single SA",
			subjects: []Model{sa1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 1, "should contain 1 serviceaccount")
				assert.Len(t, rb.Users, 0, "should contain no users")
				assert.Len(t, rb.Groups, 0, "should contain no groups")
				assert.Contains(t, rb.ServiceAccounts, sa1.UUID)
			},
		},
		{
			name:     "single user",
			subjects: []Model{u1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 0, "should contain no serviceaccounts")
				assert.Len(t, rb.Users, 1, "should contain 1 user")
				assert.Len(t, rb.Groups, 0, "should contain no groups")
				assert.Contains(t, rb.Users, u1.UUID)
			},
		},
		{
			name:     "single group",
			subjects: []Model{g1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 0, "should contain no serviceaccounts")
				assert.Len(t, rb.Users, 0, "should contain no users")
				assert.Len(t, rb.Groups, 1, "should contain 1 group")
				assert.Contains(t, rb.Groups, g1.UUID)
			},
		},
		{
			name:     "all by one",
			subjects: []Model{g1, u1, sa1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 1, "should contain 1 serviceaccount")
				assert.Len(t, rb.Users, 1, "should contain 1 user")
				assert.Len(t, rb.Groups, 1, "should contain 1 group")
				assert.Contains(t, rb.ServiceAccounts, sa1.UUID)
				assert.Contains(t, rb.Users, u1.UUID)
				assert.Contains(t, rb.Groups, g1.UUID)
			},
		},
		{
			name:     "keeps addition ordering",
			subjects: []Model{sa1, g2, u1, sa2, u2, g1},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 2, "should contain 2 serviceaccount")
				assert.Len(t, rb.Users, 2, "should contain 2 user")
				assert.Len(t, rb.Groups, 2, "should contain 2 group")

				assert.Equal(t, rb.ServiceAccounts, []ServiceAccountUUID{sa1.UUID, sa2.UUID})
				assert.Equal(t, rb.Groups, []GroupUUID{g2.UUID, g1.UUID})
				assert.Equal(t, rb.Users, []UserUUID{u1.UUID, u2.UUID})
			},
		},

		{
			name:     "ignores duplicates",
			subjects: []Model{sa1, g2, u1, sa2, u2, sa1, g1, u1, g2},
			assertions: func(t *testing.T, rb *RoleBinding) {
				assert.Len(t, rb.ServiceAccounts, 2, "should contain 2 serviceaccount")
				assert.Len(t, rb.Users, 2, "should contain 2 user")
				assert.Len(t, rb.Groups, 2, "should contain 2 group")

				assert.Equal(t, rb.ServiceAccounts, []ServiceAccountUUID{sa1.UUID, sa2.UUID})
				assert.Equal(t, rb.Groups, []GroupUUID{g2.UUID, g1.UUID})
				assert.Equal(t, rb.Users, []UserUUID{u1.UUID, u2.UUID})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := &RoleBinding{
				UUID:       uuid.New(),
				TenantUUID: ten.UUID,
				Origin:     OriginIAM,
				Subjects:   toSubjectNotations(tt.subjects...),

				ServiceAccounts: []ServiceAccountUUID{"nonsense"},
				Users:           []UserUUID{"nonsense"},
				Groups:          []GroupUUID{"nonsense"},
			}
			if err := NewRoleBindingRepository(tx).Create(rb); err != nil {
				t.Fatalf("cannot create: %v", err)
			}

			created, err := NewRoleBindingRepository(tx).GetByID(rb.UUID)
			if err != nil {
				t.Fatalf("cannot get: %v", err)
			}

			tt.assertions(t, created)
		})
	}
}

func initTestDB() (*io.MemoryStore, error) {
	schema, err := mergeSchema()
	if err != nil {
		return nil, err
	}
	return io.NewMemoryStore(schema, nil)
}

func genTenant(tx *io.MemoryStoreTxn) *Tenant {
	identifier, _ := password.Generate(10, 3, 3, false, true) // pretty random string
	ten := &Tenant{
		UUID:       uuid.New(),
		Identifier: identifier,
	}
	err := NewTenantRepository(tx).Create(ten)
	if err != nil {
		panic(fmt.Sprintf("cannot create tenant: %v", err))
	}
	return ten
}

func genUser(tx *io.MemoryStoreTxn, tid TenantUUID) *User {
	identifier, _ := password.Generate(10, 3, 3, false, true) // pretty random string
	u := &User{
		UUID:       uuid.New(),
		TenantUUID: tid,
		Origin:     OriginIAM,
		Identifier: identifier,
	}
	err := NewUserRepository(tx).Create(u)
	if err != nil {
		panic(fmt.Sprintf("cannot create user: %v", err))
	}
	return u
}

func genServiceAccount(tx *io.MemoryStoreTxn, tid TenantUUID) *ServiceAccount {
	sa := &ServiceAccount{UUID: uuid.New(), TenantUUID: tid, Origin: OriginIAM}
	err := NewServiceAccountRepository(tx).Create(sa)
	if err != nil {
		panic(fmt.Sprintf("cannot create serviceaccount: %v", err))
	}
	return sa
}

func genGroup(tx *io.MemoryStoreTxn, tid TenantUUID) *Group {
	g := &Group{UUID: uuid.New(), TenantUUID: tid, Origin: OriginIAM}
	err := NewGroupRepository(tx).Create(g)
	if err != nil {
		panic(fmt.Sprintf("cannot create group: %v", err))
	}
	return g
}
