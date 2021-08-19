package model

const MultipassGenerationNumberType = "tokengenerationnumber" // also, memdb schema name

// MultipassGenerationNumber
// This entity is a 1-to-1 relation with Multipass.
type MultipassGenerationNumber struct {
	UUID             MultipassGenerationNumberUUID `json:"uuid"` // PK == multipass uuid.
	GenerationNumber int64                         `json:"generation_number"`
}

func (t *MultipassGenerationNumber) ObjType() string {
	return MultipassGenerationNumberType
}

func (t *MultipassGenerationNumber) ObjId() string {
	return t.UUID
}
