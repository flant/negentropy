package config

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

const flantFlowConfigStorageKey = "iam.extensions.flant_flow_config"

type (
	SpecializedTeam     = string
	SpecializedRoleName = string
)

// SpecializedTeam
var (
	L1      SpecializedTeam = "L1"
	Mk8s    SpecializedTeam = "mk8s"
	Okmeter SpecializedTeam = "Okmeter"
	Devops  SpecializedTeam = "DevOps"
)

type FlantFlowConfig struct {
	FlantTenantUUID       iam_model.TenantUUID                     `json:"flant_tenant_uuid"`
	SpecificTeams         map[SpecializedTeam]model.TeamUUID       `json:"specific_teams"`
	RolesForSpecificTeams map[SpecializedTeam][]iam_model.RoleName `json:"roles_for_specific_teams"`
}

var (
	MandatorySpecificTeams             = []SpecializedTeam{L1, Mk8s, Okmeter}
	MandatoryRoleRulesForSpecificTeams = []SpecializedTeam{Devops} // TODO
)

// IsBaseConfigured returns true if prohibited to use any of client paths, returns false if allowed only use configure path
func (c *FlantFlowConfig) IsBaseConfigured() error {
	if c == nil {
		return fmt.Errorf("%w:flant flow config:nil", consts.ErrNotConfigured)
	}
	if c.FlantTenantUUID == "" {
		return fmt.Errorf("%w:flant_tenant_uuid is empty", consts.ErrNotConfigured)
	}
	if c.RolesForSpecificTeams == nil {
		return fmt.Errorf("%w:RolesForSpecificTeams:nil", consts.ErrNotConfigured)
	}

	if err := allKeysInMapOfArray(MandatoryRoleRulesForSpecificTeams, c.RolesForSpecificTeams); err != nil {
		return fmt.Errorf("%w:MandatoryRoleRulesForSpecificTeams:%s", consts.ErrNotConfigured, err.Error())
	}
	return nil
}

// IsConfigured returns true if allowed any paths
func (c *FlantFlowConfig) IsConfigured() error {
	if err := c.IsBaseConfigured(); err != nil {
		return err
	}
	if c.SpecificTeams == nil {
		return fmt.Errorf("%w:SpecificTeams:nil", consts.ErrNotConfigured)
	}
	if err := allKeysInMap(MandatorySpecificTeams, c.SpecificTeams); err != nil {
		return fmt.Errorf("%w:MandatorySpecificTeams:%s", consts.ErrNotConfigured, err.Error())
	}
	return nil
}

func allKeysInMap(ks []string, m map[string]string) error {
	for _, k := range ks {
		if _, ok := m[k]; !ok {
			return fmt.Errorf("key %q not found", k)
		}
	}
	return nil
}

func allKeysInMapOfArray(ks []string, m map[string][]string) error {
	for _, k := range ks {
		if _, ok := m[k]; !ok {
			return fmt.Errorf("key %q not found", k)
		}
	}
	return nil
}

type MutexedConfigManager struct {
	m          sync.RWMutex
	liveConfig *FlantFlowConfig
}

func (c *MutexedConfigManager) GetConfig(ctx context.Context, storage logical.Storage) (*FlantFlowConfig, error) {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.unSafeGetConfig(ctx, storage)
}

func (c *MutexedConfigManager) unSafeGetConfig(ctx context.Context, storage logical.Storage) (*FlantFlowConfig, error) {
	storedConfigEntry, err := storage.Get(ctx, flantFlowConfigStorageKey)
	if err != nil {
		return nil, err
	}
	if storedConfigEntry == nil {
		return &FlantFlowConfig{
			FlantTenantUUID:       "",
			SpecificTeams:         map[SpecializedTeam]model.TeamUUID{},
			RolesForSpecificTeams: map[SpecializedTeam][]iam_model.RoleName{},
		}, nil
	}

	var config FlantFlowConfig
	err = storedConfigEntry.DecodeJSON(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *MutexedConfigManager) unSafeSaveConfig(ctx context.Context, storage logical.Storage, config *FlantFlowConfig) (*FlantFlowConfig, error) {
	encodedValue, err := jsonutil.EncodeJSON(*config)
	if err != nil {
		return nil, err
	}

	err = storage.Put(ctx, &logical.StorageEntry{
		Key:   flantFlowConfigStorageKey,
		Value: encodedValue,
	})
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c *MutexedConfigManager) SetFlantTenantUUID(ctx context.Context, storage logical.Storage, flantTenantUUID iam_model.TenantUUID) (*FlantFlowConfig, error) {
	c.m.Lock()
	defer c.m.Unlock()
	config, err := c.unSafeGetConfig(ctx, storage)
	if err != nil {
		return nil, err
	}
	if config.FlantTenantUUID != "" {
		return nil, fmt.Errorf("flant tenant uuid already set:%s", config.FlantTenantUUID)
	}

	config.FlantTenantUUID = flantTenantUUID

	return c.unSafeSaveConfig(ctx, storage, config)
}

func (c *MutexedConfigManager) UpdateSpecificTeams(ctx context.Context, storage logical.Storage, specificTeams map[SpecializedTeam]model.TeamUUID) (*FlantFlowConfig, error) {
	c.m.Lock()
	defer c.m.Unlock()
	config, err := c.unSafeGetConfig(ctx, storage)
	if err != nil {
		return nil, err
	}
	for k, v := range specificTeams {
		config.SpecificTeams[k] = v
	}
	return c.unSafeSaveConfig(ctx, storage, config)
}

func (c *MutexedConfigManager) UpdateSpecificRoleRules(ctx context.Context, storage logical.Storage,
	teamType SpecializedTeam, roles []iam_model.RoleName) (*FlantFlowConfig, error) {
	c.m.Lock()
	defer c.m.Unlock()
	config, err := c.unSafeGetConfig(ctx, storage)
	if err != nil {
		return nil, err
	}
	config.RolesForSpecificTeams[teamType] = roles
	return c.unSafeSaveConfig(ctx, storage, config)
}
