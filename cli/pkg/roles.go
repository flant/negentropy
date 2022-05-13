package pkg

const (
	// TenantsListRole is a tenant scoped role with optional tenant
	// allows only default paths (list tenants and read token_owner)
	// it is stub role, to use it if nothing specific is needed
	TenantsListRole = "tenants.list.auth"

	// SSHOpenRole is project scoped role, definitely needs  tenant and project
	// allows UPDATE at  ssh/sign/signer with calculated allowed_parameters
	SSHOpenRole = "ssh.open"

	// ServersQueryRole is a project scoped role with optional tenant and project
	// allows READ at one of:
	// auth/flant/query_server
	// auth/flant/tenant/<tenant_uuid>/query_server
	// auth/flant/tenant/<tenant_uuid>/project/<project_uuid>/query_server
	ServersQueryRole = "servers.query"

	// TenantReadAuthRole is a tenant scoped role, definitely needs  tenant
	// allows:
	// READ at auth/flant/tenant/<tenant_uuid>
	// LIST at auth/flant/tenant/<tenant_uuid>/project
	// READ at auth/flant/tenant/<tenant_uuid>/project/+
	TenantReadAuthRole = "tenant.read.auth"

	// ServersRegisterRole is a project scoped role, definitely needs tenant and project
	// allows:
	// PUT at flant/tenant/<tenant_uuid>/project/<project_uuid>/register_server
	// PUT at flant/tenant/<tenant_uuid>/project/<project_uuid>/server/+/connection_info
	ServersRegisterRole = "servers.register"
)
