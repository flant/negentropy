package utils

import "github.com/google/uuid"

func UUID() string {
	v, err := uuid.NewRandom()
	if err != nil {
		return UUID()
	}
	return v.String()
}
