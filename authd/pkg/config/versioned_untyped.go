package config

import (
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

const ApiVersionKey = "apiVersion"
const KindKey = "kind"

type Metadata struct {
	Api     string
	Version string
	Kind    string
}

func (m Metadata) ApiVersion() string {
	return fmt.Sprintf("%s/%s", m.Api, m.Version)
}

func (m Metadata) String() string {
	return fmt.Sprintf("%s/%s/%s", m.Kind, m.Api, m.Version)
}

type VersionedUntyped struct {
	Metadata Metadata

	obj  map[string]interface{}
	data []byte
}

func (u *VersionedUntyped) DetectMetadata(data []byte) error {
	u.data = data
	err := yaml.Unmarshal(data, &u.obj)

	if err != nil {
		return fmt.Errorf("config unmarshal: %v", err)
	}

	// detect version
	u.Metadata.Api, u.Metadata.Version, err = getApiVersion(u.obj, ApiVersionKey)
	if err != nil {
		return err
	}

	// detect kind
	u.Metadata.Kind, err = getString(u.obj, KindKey)
	if err != nil {
		return err
	}

	return nil
}

func (u *VersionedUntyped) Data() []byte {
	return u.data
}

func (u *VersionedUntyped) Object() map[string]interface{} {
	return u.obj
}

// MustGetString returns non-empty string value by key
func getString(obj map[string]interface{}, key string) (value string, err error) {
	val, found := obj[key]

	if !found {
		return "", fmt.Errorf("missing '%s' key", key)
	}
	if val == nil {
		return "", fmt.Errorf("key '%s' has null value", key)
	}

	value, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("string value is expected for key '%s'", key)
	}
	if value == "" {
		return "", fmt.Errorf("key '%s' has empty value", key)
	}
	return value, nil
}

func getApiVersion(obj map[string]interface{}, key string) (string, string, error) {
	apiVersion, err := getString(obj, key)
	if err != nil {
		return "", "", err
	}

	sepIdx := strings.LastIndex(apiVersion, "/")
	if sepIdx < 1 {
		return "", apiVersion[sepIdx+1:], nil
	}

	return apiVersion[0:sepIdx], apiVersion[sepIdx+1:], nil
}
