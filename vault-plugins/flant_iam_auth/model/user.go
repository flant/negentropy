package model

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

type User = model.User

const (
	UserType = model.UserType
)

var userSchema = model.UserSchema()
