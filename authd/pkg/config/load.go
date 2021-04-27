package config

import (
	"fmt"
	"io/ioutil"
)

func LoadConfigFiles(paths []string) (*AuthdConfig, []*AuthdSocketConfig, error) {
	var authdConfig *AuthdConfig
	var authdSocketConfigs = make([]*AuthdSocketConfig, 0)

	for _, fPath := range paths {
		data, err := ioutil.ReadFile(fPath)
		if err != nil {
			return nil, nil, err
		}

		vu := new(VersionedUntyped)
		err = vu.DetectMetadata(data)
		if err != nil {
			return nil, nil, err
		}

		fmt.Printf("got %s in %s\n", vu.Metadata.String(), fPath)
		switch vu.Metadata.Kind {
		case AuthdConfigKind:
			authdConfig = &AuthdConfig{}
			err = authdConfig.Load(vu.Metadata, vu.Data())
			if err != nil {
				return nil, nil, fmt.Errorf("load %s from '%s': %v", AuthdConfigKind, fPath, err)
			}
		case AuthdSocketConfigKind:
			socketConfig := &AuthdSocketConfig{}
			err = socketConfig.Load(vu.Metadata, vu.Data())
			if err != nil {
				return nil, nil, fmt.Errorf("load %s from '%s': %v", AuthdSocketConfigKind, fPath, err)
			}
			authdSocketConfigs = append(authdSocketConfigs, socketConfig)
		}
	}

	fmt.Printf("authdConfig %v\n", authdConfig)
	fmt.Printf("authdSocketConfig %v\n", authdSocketConfigs)

	return authdConfig, authdSocketConfigs, nil
}
