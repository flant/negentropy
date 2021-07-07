package config

import (
	"fmt"
	"github.com/flant/negentropy/authd/pkg/util"
	"strings"
)

// matchDomainPattern matches srv against a pattern.
//
// 1. pattern == srv
// 2. if pattern starts with *.,  subdomain.srv is allowed
func IsDomainAllowed(server Server, serverAddr string) bool {
	if util.MatchDomainPattern(server.Domain, serverAddr) {
		return true
	}

	for _, redir := range server.AllowRedirects {
		if util.MatchDomainPattern(redir, serverAddr) {
			return true
		}
	}

	return false
}

// serverAddr может быть пустым, тогда надо найти дефолтный сервер по serverType
// если не пустой, то надо проверить его по всем серверам типа serverType
// проверка по нужна по domain и по списку allowedRedirects
func DetectServerAddr(servers []Server, serverType, serverAddr string) (string, error) {
	if serverAddr == "" {
		return GetDefaultServerAddr(servers, serverType)
	}
	var knownType bool
	serverType = strings.ToLower(serverType)
	for _, server := range servers {
		if serverType != strings.ToLower(server.Type) {
			continue
		}
		knownType = true
		if IsDomainAllowed(server, serverAddr) {
			return serverAddr, nil
		}
	}
	if !knownType {
		return "", fmt.Errorf("server type '%s' is unknown", serverType)
	}
	return "", fmt.Errorf("server '%s' is not allowed by configuration", serverAddr)
}

func GetDefaultServerAddr(servers []Server, serverType string) (string, error) {
	for _, server := range servers {
		if serverType == strings.ToLower(server.Type) {
			return server.Domain, nil
		}
	}
	return "", fmt.Errorf("server type '%s' is unknown", serverType)
}
