package model

import "time"

type UserServerPassword struct {
	Seed      []byte    `json:"seed"`
	Salt      []byte    `json:"salt"`
	ValidTill time.Time `json:"valid_till"`
}
