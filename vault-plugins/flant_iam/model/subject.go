package model

type Subjects struct {
	ServiceAccounts []ServiceAccountUUID
	Users           []UserUUID
	Groups          []GroupUUID
}

type SubjectNotation struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
