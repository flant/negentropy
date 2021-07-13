package backend

import (
	"reflect"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func Test_parseBoundRoles(t *testing.T) {
	type args struct {
		rawList interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []model.BoundRole
		wantErr bool
	}{
		{
			name:    "nil",
			want:    []model.BoundRole{},
			wantErr: false,
		},
		{
			name:    "empty list",
			args:    args{rawList: []interface{}{}},
			want:    []model.BoundRole{},
			wantErr: false,
		},
		{
			name: "valid values",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"name": "hah", "options": map[string]interface{}{}},
					map[string]interface{}{"name": "pew", "options": map[string]interface{}{"a": "b"}},
				},
			},
			want: []model.BoundRole{
				{Name: "hah", Options: map[string]interface{}{}},
				{Name: "pew", Options: map[string]interface{}{"a": "b"}},
			},
			wantErr: false,
		},
		{
			name: "error on absent role name",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"zzz": "hah", "options": map[string]interface{}{}},
					map[string]interface{}{"name": "pew", "options": map[string]interface{}{"a": "b"}},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error on absent role options",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"name": "hah", "xxxxxx": map[string]interface{}{}},
					map[string]interface{}{"name": "pew", "options": map[string]interface{}{"a": "b"}},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBoundRoles(tt.args.rawList)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBoundRoles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseBoundRoles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseSubjects(t *testing.T) {
	type args struct {
		rawList interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []model.SubjectNotation
		wantErr bool
	}{
		{
			name:    "nil",
			want:    []model.SubjectNotation{},
			wantErr: false,
		},
		{
			name:    "empty list",
			args:    args{rawList: []interface{}{}},
			want:    []model.SubjectNotation{},
			wantErr: false,
		},
		{
			name: "valid values",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"type": model.ServiceAccountType, "id": "said"},
					map[string]interface{}{"type": model.GroupType, "id": "gid"},
					map[string]interface{}{"type": model.UserType, "id": "uid"},
				},
			},
			want: []model.SubjectNotation{
				{Type: model.ServiceAccountType, ID: "said"},
				{Type: model.GroupType, ID: "gid"},
				{Type: model.UserType, ID: "uid"},
			},
			wantErr: false,
		},
		{
			name: "error on absent type",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"type": model.ServiceAccountType, "id": "said"},
					map[string]interface{}{"___": model.GroupType, "id": "gid"},
					map[string]interface{}{"type": model.UserType, "id": "uid"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error on absent id",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"type": model.ServiceAccountType, "id": "said"},
					map[string]interface{}{"type": model.GroupType, "___": "gid"},
					map[string]interface{}{"type": model.UserType, "id": "uid"},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSubjects(tt.args.rawList)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSubjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSubjects() got = %v, want %v", got, tt.want)
			}
		})
	}
}
