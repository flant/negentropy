package memdb

import (
	"math/rand"
	"time"
)

type Archivable interface {
	Archive(archiveMark ArchiveMark)
	Restore()
	Archived() bool
	GetArchiveMark() ArchiveMark
	Equals(other ArchiveMark) bool
}

type ArchiveMark struct {
	Timestamp UnixTime `json:"archiving_timestamp"`
	Hash      int64    `json:"archiving_hash"`
}

func (a *ArchiveMark) Archive(archiveMark ArchiveMark) {
	a.Timestamp = archiveMark.Timestamp
	a.Hash = archiveMark.Hash
}

func (a *ArchiveMark) Restore() {
	a.Timestamp = 0
	a.Hash = 0
}

func (a *ArchiveMark) Archived() bool {
	return a.Timestamp != 0
}

func (a *ArchiveMark) GetArchiveMark() ArchiveMark {
	if a == nil {
		return ArchiveMark{}
	}
	return *a
}

func (a *ArchiveMark) Equals(other ArchiveMark) bool {
	return a.Timestamp == other.Timestamp && a.Hash == other.Hash
}

func NewArchiveMark() ArchiveMark {
	archivingTime := time.Now().Unix()
	return ArchiveMark{
		Timestamp: archivingTime,
		Hash:      rand.Int63n(archivingTime),
	}
}

var ActiveRecordMark = ArchiveMark{}
