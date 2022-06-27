package model

import (
	"encoding/json"
	"fmt"
)

// ServicePack names
var (
	L1                      ServicePackName = "l1_service_pack"
	DevOps                  ServicePackName = "devops_service_pack"
	Mk8s                    ServicePackName = "mk8s_service_pack"
	Deckhouse               ServicePackName = "deckhouse_service_pack"
	Okmeter                 ServicePackName = "okmeter_service_pack"
	Consulting              ServicePackName = "consulting_service_pack"
	InternalProject         ServicePackName = "internal_project_service_pack"
	AllowedServicePackNames                 = []interface{}{L1, DevOps, Mk8s, Deckhouse, Okmeter, Consulting, InternalProject}
	ServicePackNames                        = map[ServicePackName]struct{}{L1: {}, DevOps: {}, Mk8s: {}, Deckhouse: {}, Okmeter: {}, Consulting: {}, InternalProject: {}}
)

type ServicePackCFG interface{}

type DevopsServicePackCFG struct {
	DevopsTeam TeamUUID `json:"devops_team"`
}
type L1ServicePackCFG struct{}

type Mk8sServicePackCFG struct{}

type DeckhouseServicePackCFG struct{}

type OkmeterServicePackCFG struct{}

type ConsultingServicePackCFG struct{}

type InternalProjectServicePackCFG struct {
	Team TeamUUID `json:"team"`
}

func ParseServicePacks(servicePacks map[ServicePackName]interface{}) (map[ServicePackName]ServicePackCFG, error) {
	result := map[ServicePackName]ServicePackCFG{}
	for k, v := range servicePacks {
		bytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if string(bytes) == "null" {
			result[k] = nil
			continue
		}
		switch k {
		case L1:
			cfg := L1ServicePackCFG{}
			err = json.Unmarshal(bytes, &cfg)
			result[k] = cfg
		case DevOps:
			cfg := DevopsServicePackCFG{}
			err = json.Unmarshal(bytes, &cfg)
			result[k] = cfg
		case Mk8s:
			cfg := &Mk8sServicePackCFG{}
			err = json.Unmarshal(bytes, &cfg)
			result[k] = cfg
		case Deckhouse:
			cfg := &DeckhouseServicePackCFG{}
			err = json.Unmarshal(bytes, &cfg)
			result[k] = cfg
		case Okmeter:
			cfg := &OkmeterServicePackCFG{}
			err = json.Unmarshal(bytes, &cfg)
			result[k] = cfg
		case Consulting:
			cfg := &ConsultingServicePackCFG{}
			err = json.Unmarshal(bytes, &cfg)
			result[k] = cfg
		case InternalProject:
			cfg := &InternalProjectServicePackCFG{}
			err = json.Unmarshal(bytes, &cfg)
			result[k] = cfg
		}
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func TryGetDevopsCFG(servicePacks map[ServicePackName]ServicePackCFG) (cfg *DevopsServicePackCFG, err error, found bool) {
	rawCFG, ok := servicePacks[DevOps]
	if !ok {
		return nil, nil, false
	}
	c, ok := rawCFG.(DevopsServicePackCFG)
	if !ok {
		return nil, fmt.Errorf("wrong cfg format: expected  DevopsServicePackCFG, got: %T", rawCFG), true
	}
	return &c, nil, true
}

func TryGetInternalProjectCFG(servicePacks map[ServicePackName]ServicePackCFG) (cfg *InternalProjectServicePackCFG, err error, found bool) {
	rawCFG, ok := servicePacks[InternalProject]
	if !ok {
		return nil, nil, false
	}
	c, ok := rawCFG.(InternalProjectServicePackCFG)
	if !ok {
		return nil, fmt.Errorf("wrong cfg format: expected  InternalProjectServicePackCFG, got: %T", rawCFG), true
	}
	return &c, nil, true
}
