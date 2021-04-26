package backend

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestUUIDParam(t *testing.T) {

	uuid := genUUID()

	tests := []struct {
		name        string
		input       string
		expectMatch string
	}{
		{
			name:        "lowercase uuid",
			input:       strings.ToLower(uuid),
			expectMatch: strings.ToLower(uuid),
		},
		{
			name:        "uppercase uuid",
			input:       strings.ToUpper(uuid),
			expectMatch: strings.ToUpper(uuid),
		},
		{
			name:        "uuid + extra character",
			input:       uuid + "x",
			expectMatch: "",
		},

		{
			name:        "uuid - characters",
			input:       uuid[:11],
			expectMatch: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name := "name"

			pattern := "^" + UUIDParam(name) + "$"
			re, err := regexp.Compile(pattern)
			if err != nil {
				t.Fatalf("UUID regex does not compile: %v", err)
			}

			match := findMatch(re, name, test.input)
			if match != test.expectMatch {
				t.Fatalf("expected input %q to match %q", test.expectMatch, match)
			}

		})
	}
}

func TestOptionalUUIDParam(t *testing.T) {

	uuid := genUUID()

	tests := []struct {
		name        string
		input       string
		expectMatch string
	}{
		{
			name:        "lowercase uuid",
			input:       "/" + strings.ToLower(uuid),
			expectMatch: strings.ToLower(uuid),
		},
		{
			name:        "uppercase uuid",
			input:       "/" + strings.ToUpper(uuid),
			expectMatch: strings.ToUpper(uuid),
		},
		{
			name:        "empty string should match",
			input:       "",
			expectMatch: "",
		},
		{
			name:        "invalid uuid: extra character",
			input:       "/" + uuid + "x",
			expectMatch: "",
		},

		{
			name:        "invalid uuid: not enough characters",
			input:       "/" + uuid[:11],
			expectMatch: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name := "name"

			pattern := OptionalUUIDParam(name)
			re, err := regexp.Compile(pattern)
			if err != nil {
				t.Fatalf("UUID regex does not compile: %v", err)
			}

			match := findMatch(re, name, test.input)
			if match != test.expectMatch {
				t.Fatalf("expected input %q to match %q", test.expectMatch, match)
			}

		})
	}

}

func findMatch(re *regexp.Regexp, name, input string) string {
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return ""
	}
	fmt.Printf("findMatch matches: %v\n", matches)
	for i, capName := range re.SubexpNames() {
		if capName == name {
			return matches[i]
		}
	}
	return ""
}
