package types

type User struct {
	Name       string `json:"name,omitempty" db:"name"`
	Uid        uint   `json:"uid,omitempty" db:"uid"`
	Gid        uint   `json:"gid,omitempty" db:"gid"`
	Gecos      string `json:"gecos,omitempty" db:"gecos"`
	HomeDir    string `json:"home_dir,omitempty" db:"homedir"`
	Shell      string `json:"shell,omitempty" db:"shell"`
	HashedPass string `json:"hashed_pass,omitempty" db:"hashed_pass"`
	Principal  string `json:"principal"`
}

type Group struct {
	Name string `json:"name,omitempty" db:"name"`
	Gid  uint   `json:"gid,omitempty" db:"gid"`
}

type UsersWithGroups struct {
	Users  []User  `json:"users,omitempty"`
	Groups []Group `json:"groups,omitempty"`
}
