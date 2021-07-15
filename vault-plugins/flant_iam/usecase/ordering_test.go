package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func TestSubjectsFetcher_Fetch(t *testing.T) {
	type fields struct {
		serviceAccountRepo RawGetter
		userRepo           RawGetter
		groupRepo          RawGetter
	}
	alwaysExisting := fields{
		serviceAccountRepo: &AlwaysExistingServiceAccounts{},
		userRepo:           &AlwaysExistingUsers{},
		groupRepo:          &AlwaysExistingGroups{},
	}

	type args struct {
		subjects []model.SubjectNotation
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *model.Subjects
		wantErr bool
	}{
		{
			name:   "nil input results in empty slices",
			fields: alwaysExisting,
			args:   args{subjects: nil},
			want: &model.Subjects{
				ServiceAccounts: []string{},
				Users:           []string{},
				Groups:          []string{},
			},
			wantErr: false,
		},
		{
			name:   "deduplicates and sorts",
			fields: alwaysExisting,
			args: args{subjects: []model.SubjectNotation{
				{Type: model.ServiceAccountType, ID: "sa1"},
				{Type: model.ServiceAccountType, ID: "sa2"},
				{Type: model.ServiceAccountType, ID: "sa3"},
				{Type: model.ServiceAccountType, ID: "sa2"},
				{Type: model.ServiceAccountType, ID: "sa1"},
				{Type: model.UserType, ID: "u1"},
				{Type: model.UserType, ID: "u2"},
				{Type: model.UserType, ID: "u3"},
				{Type: model.UserType, ID: "u2"},
				{Type: model.UserType, ID: "u1"},
				{Type: model.GroupType, ID: "g1"},
				{Type: model.GroupType, ID: "g2"},
				{Type: model.GroupType, ID: "g3"},
				{Type: model.GroupType, ID: "g2"},
				{Type: model.GroupType, ID: "g1"},
			}},
			want: &model.Subjects{
				ServiceAccounts: []string{"sa1", "sa2", "sa3"},
				Users:           []string{"u1", "u2", "u3"},
				Groups:          []string{"g1", "g2", "g3"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &SubjectsFetcher{
				serviceAccountRepo: tt.fields.serviceAccountRepo,
				userRepo:           tt.fields.userRepo,
				groupRepo:          tt.fields.groupRepo,
			}

			got, err := f.Fetch(tt.args.subjects)

			if (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

type AlwaysExistingServiceAccounts struct{}

func (mock *AlwaysExistingServiceAccounts) GetRawByID(id string) (interface{}, error) {
	return &model.ServiceAccount{UUID: id}, nil
}

type AlwaysExistingUsers struct{}

func (mock *AlwaysExistingUsers) GetRawByID(id string) (interface{}, error) {
	return &model.User{UUID: id}, nil
}

type AlwaysExistingGroups struct{}

func (mock *AlwaysExistingGroups) GetRawByID(id string) (interface{}, error) {
	return &model.Group{UUID: id}, nil
}
