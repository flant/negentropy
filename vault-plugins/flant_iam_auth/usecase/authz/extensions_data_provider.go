package authz

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ExtensionsDataProvider interface {
	CollectExtensionsData(extensions []string, subject model.Subject, claim model.RoleClaim) (map[string]interface{}, error)
}

type ExtensionDataProvider interface {
	GetExtensionData(subject model.Subject, claim model.RoleClaim) (map[string]interface{}, error)
}

type extensionDataProvider struct {
	providers map[string]ExtensionDataProvider
}

func (p *extensionDataProvider) CollectExtensionsData(extensions []string, subject model.Subject, claim model.RoleClaim) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for _, ext := range extensions {
		if provider, ok := p.providers[ext]; ok {
			extData, err := provider.GetExtensionData(subject, claim)
			if err != nil {
				return nil, fmt.Errorf("collecting extension_data for extension '%s': %w", ext, err)
			}
			for k, v := range extData {
				result[k] = v
			}
		} else {
			return nil, fmt.Errorf("ExtensionDataProvider for extension '%s' not found", ext)
		}
	}
	return result, nil
}

func NewExtensionsDataProvider(db *io.MemoryStoreTxn) ExtensionsDataProvider {
	return &extensionDataProvider{
		providers: map[string]ExtensionDataProvider{
			"server_access": extension_server_access.NewDataProvider(db),
		},
	}
}
