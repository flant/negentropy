package model

import iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const ClientType = "client" // also, memdb schema name

type Client = iam_model.Tenant
