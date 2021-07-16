package model

type Members struct {
	ServiceAccounts []ServiceAccountUUID
	Users           []UserUUID
	Groups          []GroupUUID
}

type MemberNotation struct {
	Type string `json:"type"`
	UUID string `json:"uuid"`
}
