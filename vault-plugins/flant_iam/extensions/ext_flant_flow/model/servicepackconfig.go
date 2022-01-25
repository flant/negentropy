package model

import "encoding/json"

// ServicePack names
var (
	L1                      ServicePackName = "L1"
	DevOps                  ServicePackName = "DevOps"
	Mk8s                    ServicePackName = "mk8s"
	Deckhouse               ServicePackName = "Deckhouse"
	Okmeter                 ServicePackName = "Okmeter"
	Consulting              ServicePackName = "Consulting"
	AllowedServicePackNames                 = []interface{}{L1, DevOps, Mk8s, Deckhouse, Okmeter, Consulting}
	ServicePackNames                        = map[ServicePackName]struct{}{L1: {}, DevOps: {}, Mk8s: {}, Deckhouse: {}, Okmeter: {}, Consulting: {}}
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
		}
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
