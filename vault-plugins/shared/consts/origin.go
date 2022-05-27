package consts

type ObjectOrigin string

const (
	OriginFlantFlowPredefined              = "flant_flow_predefined"
	OriginIAM                 ObjectOrigin = "iam"
	OriginServerAccess        ObjectOrigin = "server_access"
	OriginFlantFlow           ObjectOrigin = "flant_flow"
	OriginAUTH                ObjectOrigin = "auth"
)

func ValidateOrigin(origin ObjectOrigin) error {
	if origin == OriginIAM ||
		origin == OriginServerAccess ||
		origin == OriginFlantFlow ||
		origin == OriginAUTH {
		return nil
	}
	return ErrBadOrigin
}
