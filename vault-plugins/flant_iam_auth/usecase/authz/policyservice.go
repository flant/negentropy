package authz

import (
	"fmt"
	"strings"
)

// Rule represents atomic rule for vault policy
type Rule struct {
	Path   string
	Create bool
	Update bool
	Read   bool
	Delete bool
	List   bool
}

// String represents Rule in vault form
func (r *Rule) String() string {
	caps := []string{}
	if r.Create {
		caps = append(caps, "create")
	}
	if r.Update {
		caps = append(caps, "update")
	}
	if r.Read {
		caps = append(caps, "read")
	}
	if r.Delete {
		caps = append(caps, "delete")
	}
	if r.List {
		caps = append(caps, "list")
	}
	capsString := fmt.Sprintf("%#v", caps)         // []string{"create", "update", "read", "delete", "list"}
	capsString = capsString[9 : len(capsString)-1] // "create", "update", "read", "delete", "list"
	return fmt.Sprintf("path \"%s\" {\n   capabilities = [%s]\n}", r.Path, capsString)
}

type Policy struct {
	Name  string // unique name
	Rules []Rule
}

// PolicyRules collects all rules into form using to passed to vault
func (p *Policy) PolicyRules() string {
	builder := strings.Builder{}
	firstElem := true
	for _, r := range p.Rules {
		if firstElem {
			firstElem = false
		} else {
			builder.WriteString("\n\n")
		}
		builder.WriteString(r.String())
	}
	return builder.String()
}
