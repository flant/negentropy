package util

import (
	"regexp"
	"strings"
)

func MatchDomainPattern(pattern, in string) bool {
	// Check if pattern
	if !strings.Contains(pattern, "*") {
		// fallback
		return in == pattern
	}

	pattern = strings.Replace(pattern, ".", "\\.", -1)
	pattern = strings.Replace(pattern, "*", ".*", -1)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(in)
}

//patternDomain := strings.TrimPrefix(pattern, "*.")
//srvParts := strings.SplitN(srv, ".", 2)
//if len(srvParts) == 2 && patternDomain == srvParts[1] {
//	return true
//}
