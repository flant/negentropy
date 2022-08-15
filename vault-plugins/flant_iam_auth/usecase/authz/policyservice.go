package authz

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Rule represents atomic rule for vault policy
type Rule struct {
	Path               string
	AllowedParameters  map[string][]string
	RequiredParameters []string
	Create             bool
	Update             bool
	Read               bool
	Delete             bool
	List               bool
}

// String represents Rule in vault form
func (r *Rule) String() string {
	capabilitiesString := buildCapabilities(*r)
	requiredParametersString := buildRequiredParametersString(r.RequiredParameters)
	allowedParametersString := buildAllowedParametersString(r.AllowedParameters)
	return fmt.Sprintf("path \"%s\" {\n%s%s%s}", r.Path,
		capabilitiesString, requiredParametersString, allowedParametersString)
}

func buildCapabilities(r Rule) string {
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
	return buildStringSlice("   capabilities", caps, false)
}

func buildAllowedParametersString(allowedParameters map[string][]string) string {
	if len(allowedParameters) == 0 {
		return ""
	}
	builder := strings.Builder{}
	builder.WriteString("   allowed_parameters = {\n")
	for name, params := range allowedParameters {
		builder.WriteString("      " + buildStringSlice("\""+name+"\"", params, false))
	}
	builder.WriteString("   }\n")
	return builder.String()
}

func buildRequiredParametersString(requiredParameters []string) string {
	return buildStringSlice("   required_parameters", requiredParameters, true)
}

func buildStringSlice(name string, slice []string, skipEmpty bool) string {
	if len(slice) == 0 && skipEmpty {
		return ""
	}
	elemsString := ""
	if len(slice) > 0 {
		elemsString = "\"" + strings.Join(slice, "\", \"") + "\""
	}
	return fmt.Sprintf("%s = [%s]\n", name, elemsString)
}

type VaultPolicy struct {
	Name  string `json:"name,omitempty"`
	Rules []Rule `json:"rules"`
}

const policyNameDateFormat = "2006-01-02t15:04:05"

// AddValidTillToName adds suffix _valid_till_timestamp at UTC
func (p *VaultPolicy) AddValidTillToName(validTill time.Time) {
	p.Name = p.Name + "_till_" + validTill.UTC().Format(policyNameDateFormat)
}

// IsVaultPolicyOverdue returns true if suffix  _valid_till_timestamp contains past time
// returns error if can't parse timestamp
func IsVaultPolicyOverdue(policyName string) (bool, error) {
	items := strings.Split(policyName, "_till_")
	if len(items) == 1 {
		return false, nil
	}
	if len(items) > 2 {
		return false, fmt.Errorf("double '_till_' at policy_name:%s", policyName)
	}
	timeStamp, err := time.Parse(policyNameDateFormat, items[1])
	if err != nil {
		return false, fmt.Errorf("parsing policy_name=%s :%w", policyName, err)
	}
	return timeStamp.Before(time.Now().UTC()), nil
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
		Path               string              `json:"path"`
		Capabilities       []string            `json:"capabilities"`
		AllowedParameters  map[string][]string `json:"allowed_parameters"`
		RequiredParameters []string            `json:"required_parameters"`
	}{
		Path:               r.Path,
		Capabilities:       caps,
		AllowedParameters:  r.AllowedParameters,
		RequiredParameters: r.RequiredParameters,
	})
}

func (r *Rule) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Path               string              `json:"path"`
		Capabilities       []string            `json:"capabilities"`
		AllowedParameters  map[string][]string `json:"allowed_parameters"`
		RequiredParameters []string            `json:"required_parameters"`
	}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	r.Path = tmp.Path
	r.AllowedParameters = tmp.AllowedParameters
	r.RequiredParameters = tmp.RequiredParameters
	for _, c := range tmp.Capabilities {
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
