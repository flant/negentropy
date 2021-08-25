package model

type SafeProject struct {
	UUID       string `json:"uuid"` // PK
	TenantUUID string `json:"tenant_uuid"`
	Version    string `json:"resource_version"`
}

type Project struct {
	UUID       string `json:"uuid"` // PK
	TenantUUID string `json:"tenant_uuid"`
	Version    string `json:"resource_version"`
	Identifier string `json:"identifier"`
}

type SafeTenant struct {
	UUID    string `json:"uuid"`
	Version string `json:"version"`
}

type SafeServer struct {
	UUID        string `json:"uuid"`
	Version     string `json:"resource_version"`
	ProjectUUID string `json:"project_uuid"`
	TenantUUID  string `json:"tenant_uuid"`
}

type Server struct {
	UUID        string `json:"uuid"`
	Identifier  string `json:"identifier"`
	Version     string `json:"resource_version"`
	ProjectUUID string `json:"project_uuid"`
	TenantUUID  string `json:"tenant_uuid"`
}

type User struct {
	UUID       string `json:"uuid"` // PK
	TenantUUID string `json:"tenant_uuid"`

	Origin string `json:"origin"`

	Identifier     string `json:"identifier"`
	FullIdentifier string `json:"full_identifier"` // calculated <identifier>@<tenant_identifier>

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`

	Email            string   `json:"email"`
	AdditionalEmails []string `json:"additional_emails"`

	MobilePhone      string   `json:"mobile_phone"`
	AdditionalPhones []string `json:"additional_phones"`
}

type ServiceAccount struct {
	UUID           string   `json:"uuid"` // PK
	TenantUUID     string   `json:"tenant_uuid"`
	BuiltinType    string   `json:"-"`
	Identifier     string   `json:"identifier"`
	FullIdentifier string   `json:"full_identifier"`
	CIDRs          []string `json:"allowed_cidrs"`

	Origin string `json:"origin"`
}
