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
	tenantsTimestamps  map[iam.TenantUUID]time.Time
	projectsTimestamps map[iam.ProjectUUID]time.Time
	serversTimestamps  map[ext.ServerUUID]time.Time
	// ttl of entity
	ttl time.Duration
}

func (c *Cache) ClearOverdue() {
	tooLongAgo := time.Now().Add(-c.ttl)
	for id, lastAccess := range c.tenantsTimestamps {
		if lastAccess.Before(tooLongAgo) {
			delete(c.Tenants, id)
			delete(c.tenantsTimestamps, id)
		}
	}
	for id, lastAccess := range c.projectsTimestamps {
		if lastAccess.Before(tooLongAgo) {
			delete(c.Projects, id)
			delete(c.projectsTimestamps, id)
		}
	}
	for id, lastAccess := range c.serversTimestamps {
		if lastAccess.Before(tooLongAgo) {
			delete(c.Servers, id)
			delete(c.serversTimestamps, id)
		}
	}
}

func (c *Cache) SaveToFile(path string) error {
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
	result.ttl = ttl
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
		} else if err != nil {
			return nil, fmt.Errorf("tryReadCacheFromFile, accessing dir: %w", err)
		}
		cache := Cache{ttl: ttl}
		cache.initializeIfEmpty()
		return &cache, nil
	} else if err != nil {
		return nil, fmt.Errorf("tryReadCacheFromFile, accessing cache file: %w", err)
	}
	return ReadFromFile(filePath, ttl)
}

func (c *Cache) Update(freshList ServerList) {
	c.initializeIfEmpty()
	now := time.Now()
	for k, v := range freshList.Tenants {
		c.Tenants[k] = v
		c.tenantsTimestamps[k] = now
	}
	for k, v := range freshList.Projects {
		c.Projects[k] = v
		c.projectsTimestamps[k] = now
	}
	for k, v := range freshList.Servers {
		c.Servers[k] = v
		c.serversTimestamps[k] = now
	}
	c.ClearOverdue()
}

func (c *Cache) initializeIfEmpty() {
	if c.Projects == nil {
		c.Projects = map[iam.ProjectUUID]iam.Project{}
	}
	if c.Tenants == nil {
		c.Tenants = map[iam.TenantUUID]iam.Tenant{}
	}
	if c.Servers == nil {
		c.Servers = map[ext.ServerUUID]ext.Server{}
	}
	if c.tenantsTimestamps == nil {
		c.tenantsTimestamps = map[iam.TenantUUID]time.Time{}
	}
	if c.projectsTimestamps == nil {
		c.projectsTimestamps = map[iam.ProjectUUID]time.Time{}
	}
	if c.serversTimestamps == nil {
		c.serversTimestamps = map[ext.ServerUUID]time.Time{}
	}
}
