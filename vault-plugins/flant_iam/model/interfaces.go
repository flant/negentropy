package model

type Marshaller interface {
	Marshal(bool) ([]byte, error)
	Unmarshal([]byte) error
}
