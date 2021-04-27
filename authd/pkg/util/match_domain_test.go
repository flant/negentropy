package util

import "testing"

func Test_MatchDomain(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		in      string
		result  bool
	}{
		{
			"exact match",
			"domain.com",
			"domain.com",
			true,
		},
		{
			"not match",
			"domain.com",
			"asd.domain.com",
			false,
		},
		{
			"glob match",
			"*.domain.com",
			"sub.domain.com",
			true,
		},
		{
			"glob not match",
			"*.domain.com",
			"domain.com",
			false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := MatchDomainPattern(tt.pattern, tt.in)
			if res == tt.result {
				return
			}
			if tt.result {
				t.Fatalf("'%s' should match pattern '%s'", tt.in, tt.pattern)
			} else {
				t.Fatalf("'%s' should not match pattern '%s'", tt.in, tt.pattern)
			}
		})
	}
}
