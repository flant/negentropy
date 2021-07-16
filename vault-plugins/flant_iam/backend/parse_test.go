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

func Test_parseMembers(t *testing.T) {
	type args struct {
		rawList interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []model.MemberNotation
		wantErr bool
	}{
		{
			name:    "nil",
			want:    []model.MemberNotation{},
			wantErr: false,
		},
		{
			name:    "empty list",
			args:    args{rawList: []interface{}{}},
			want:    []model.MemberNotation{},
			wantErr: false,
		},
		{
			name: "valid values",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"type": model.ServiceAccountType, "uuid": "said"},
					map[string]interface{}{"type": model.GroupType, "uuid": "gid"},
					map[string]interface{}{"type": model.UserType, "uuid": "uid"},
				},
			},
			want: []model.MemberNotation{
				{Type: model.ServiceAccountType, UUID: "said"},
				{Type: model.GroupType, UUID: "gid"},
				{Type: model.UserType, UUID: "uid"},
			},
			wantErr: false,
		},
		{
			name: "error on absent type",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"type": model.ServiceAccountType, "uuid": "said"},
					map[string]interface{}{"___": model.GroupType, "uuid": "gid"},
					map[string]interface{}{"type": model.UserType, "uuid": "uid"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error on absent id",
			args: args{
				rawList: []interface{}{
					map[string]interface{}{"type": model.ServiceAccountType, "uuid": "said"},
					map[string]interface{}{"type": model.GroupType, "___": "gid"},
					map[string]interface{}{"type": model.UserType, "uuid": "uid"},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMembers(tt.args.rawList)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMembers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMembers() got = %v, want %v", got, tt.want)
			}
		})
	}
}
