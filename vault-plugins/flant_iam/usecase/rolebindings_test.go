package usecase

/*


import "github.com/stretchr/testify/assert"

func Test_RoleBindingOnCreationSubjectsCalculation(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, projectFixture, roleFixture,
		roleBindingFixture).Txn(true)
	ten := &tenant1
	sa1 := &sa1
	sa2 := &sa2
	u1 := &user1
	u2 := &user2
	g1 := &group1
	g2 := &group2

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

*/
