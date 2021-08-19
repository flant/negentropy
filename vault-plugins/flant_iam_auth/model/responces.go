package model

type ProjectIdentifiers struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
}

type TenantIdentifiers struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
}

type Server struct {
	UUID        string `json:"uuid"`
	Identifier  string `json:"identifier"`
	Version     string `json:"resource_version"`
	ProjectUUID string `json:"project_uuid"`
	TenantUUID  string `json:"tenant_uuid"`
}
