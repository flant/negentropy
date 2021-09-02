package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type ServerList struct {
	Tenants  map[iam.TenantUUID]iam.Tenant
	Projects map[iam.ProjectUUID]iam.Project
	Servers  map[ext.ServerUUID]ext.Server
}

func SaveToFile(serverList ServerList, path string) error {
	data, err := json.Marshal(serverList)
	if err != nil {
		return fmt.Errorf("SaveToFile: %w", err)
	}
	err = ioutil.WriteFile(path, data, 0o644)
	if err != nil {
		return fmt.Errorf("SaveToFile: %w", err)
	}
	return nil
}

func ReadFromFile(path string) (*ServerList, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ReadFromFile: %w", err)
	}
	var result ServerList
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("ReadFromFile: %w", err)
	}
	return &result, nil
}

func TryReadCacheFromFile(filePath string) (*ServerList, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		dirPath := path.Dir(filePath)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			err = os.MkdirAll(dirPath, 0o644) // Create path
			if err != nil {
				return nil, fmt.Errorf("tryReadCacheFromFile, creating dirs: %w", err)
			}
		}
		return &ServerList{
			Tenants:  map[iam.TenantUUID]iam.Tenant{},
			Projects: map[iam.ProjectUUID]iam.Project{},
			Servers:  map[ext.ServerUUID]ext.Server{},
		}, nil
	}
	return ReadFromFile(filePath)
}

func UpdateServerListCacheWithFreshValues(cache ServerList, freshList ServerList) ServerList {
	for k, v := range freshList.Tenants {
		cache.Tenants[k] = v
	}
	for k, v := range freshList.Projects {
		cache.Projects[k] = v
	}
	for k, v := range freshList.Servers {
		cache.Servers[k] = v
	}
	return cache
}
