package authz

import (
	"encoding/json"
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

type VaultPolicy struct {
	Name  string `json:"name,omitempty"`
	Rules []Rule `json:"rules"`
}

// PolicyRules collects all rules into form used to passed to vault
func (p *VaultPolicy) PolicyRules() string {
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

type VaultPolicyManager interface {
	CreatePolicy(VaultPolicy) error
	RunGarbageCollector()
}

func (r *Rule) MarshalJSON() ([]byte, error) {
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
	return json.Marshal(&struct {
		Path         string   `json:"path"`
		Capabilities []string `json:"capabilities"`
	}{
		Path:         r.Path,
		Capabilities: caps,
	})
}

func (r *Rule) UnmarshalJSON(data []byte) error {
	var raw struct {
		Path         string   `json:"path"`
		Capabilities []string `json:"capabilities"`
	}
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	r.Path = raw.Path
	for _, c := range raw.Capabilities {
		switch c {
		case "create":
			r.Create = true
		case "update":
			r.Update = true
		case "read":
			r.Read = true
		case "delete":
			r.Delete = true
		case "list":
			r.List = true
		}
	}
	return nil
}
