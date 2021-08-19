package jwt

import "github.com/hashicorp/cap/jwt"

func ToAlg(a []string) []jwt.Alg {
	alg := make([]jwt.Alg, len(a))
	for i, e := range a {
		alg[i] = jwt.Alg(e)
	}
	return alg
}
