package vault

// ServerRole is project scoped role, definitely needs  tenant and project
// allows LIST at  flant/tenant/<tenant_uuid>/project/<project_uuid>/server/<server_uuid>/posix_users
// needs server_uuid be passed tr
const ServerRole = "server"
