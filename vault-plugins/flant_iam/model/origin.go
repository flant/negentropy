package model

type ObjectOrigin string

const (
	OriginIAM          ObjectOrigin = "iam"
	OriginServerAccess ObjectOrigin = "server_access"
)
