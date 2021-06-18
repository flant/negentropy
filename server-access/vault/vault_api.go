package vault

import "github.com/hashicorp/vault/api"

type Logical interface {
	Read(path string) (*api.Secret, error)
	ReadWithData(path string, data map[string][]string) (*api.Secret, error)
	List(path string) (*api.Secret, error)
	Write(path string, data map[string]interface{}) (*api.Secret, error)
	WriteBytes(path string, data []byte) (*api.Secret, error)
	Delete(path string) (*api.Secret, error)
	DeleteWithData(path string, data map[string][]string) (*api.Secret, error)
	Unwrap(wrappingToken string) (*api.Secret, error)
}
