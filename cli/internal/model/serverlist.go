package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type ServerList struct {
	Tenants  map[iam.TenantUUID]iam.Tenant
	Projects map[iam.ProjectUUID]iam.Project
	Servers  map[ext.ServerUUID]ext.Server
}

type Cache struct {
	ServerList
	// last vault access to entity
	TenantsTimestamps  map[iam.TenantUUID]time.Time
	ProjectsTimestamps map[iam.ProjectUUID]time.Time
	ServersTimestamps  map[ext.ServerUUID]time.Time
	// ttl of entity
	TTL time.Duration
}

func (c Cache) ClearOverdue() {
	now := time.Now()
	for id, lastAccess := range c.TenantsTimestamps {
		if now.After(lastAccess.Add(c.TTL)) {
			delete(c.Tenants, id)
			delete(c.TenantsTimestamps, id)
		}
	}
	for id, lastAccess := range c.ProjectsTimestamps {
		if now.After(lastAccess.Add(c.TTL)) {
			delete(c.Projects, id)
			delete(c.ProjectsTimestamps, id)
		}
	}
	for id, lastAccess := range c.ServersTimestamps {
		if now.After(lastAccess.Add(c.TTL)) {
			delete(c.Servers, id)
			delete(c.ServersTimestamps, id)
		}
	}
}

func (c Cache) SaveToFile(path string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("SaveToFile: %w", err)
	}
	err = ioutil.WriteFile(path, data, 0o644)
	if err != nil {
		return fmt.Errorf("SaveToFile: %w", err)
	}
	return nil
}

func ReadFromFile(path string, ttl time.Duration) (*Cache, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ReadFromFile: %w", err)
	}
	var result Cache
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("ReadFromFile: %w", err)
	}
	result.TTL = ttl
	return &result, nil
}

func TryReadCacheFromFile(filePath string, ttl time.Duration) (*Cache, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		dirPath := path.Dir(filePath)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			err = os.MkdirAll(dirPath, 0o644) // Create path
			if err != nil {
				return nil, fmt.Errorf("tryReadCacheFromFile, creating dirs: %w", err)
			}
		}
		return &Cache{
			ServerList: ServerList{
				Tenants:  map[iam.TenantUUID]iam.Tenant{},
				Projects: map[iam.ProjectUUID]iam.Project{},
				Servers:  map[ext.ServerUUID]ext.Server{},
			},
			TenantsTimestamps:  map[iam.TenantUUID]time.Time{},
			ProjectsTimestamps: map[iam.ProjectUUID]time.Time{},
			ServersTimestamps:  map[ext.ServerUUID]time.Time{},
			TTL:                ttl,
		}, nil
	}
	return ReadFromFile(filePath, ttl)
}

func (c Cache) Update(freshList ServerList) {
	now := time.Now()
	for k, v := range freshList.Tenants {
		c.Tenants[k] = v
		c.TenantsTimestamps[k] = now
	}
	for k, v := range freshList.Projects {
		c.Projects[k] = v
		c.ProjectsTimestamps[k] = now
	}
	for k, v := range freshList.Servers {
		c.Servers[k] = v
		c.ServersTimestamps[k] = now
	}
	c.ClearOverdue()
}
