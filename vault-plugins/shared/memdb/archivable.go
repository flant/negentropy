package memdb

type Archivable interface {
	Archive(timeStamp UnixTime, archivingHash int64)
	Restore()
	Archived() bool
	ArchiveMarks() (timeStamp UnixTime, archivingHash int64)
}

type ArchivableImpl struct {
	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (a *ArchivableImpl) Archive(timeStamp UnixTime, hash int64) {
	a.ArchivingTimestamp = timeStamp
	a.ArchivingHash = hash
}

func (a *ArchivableImpl) Restore() {
	a.ArchivingTimestamp = 0
	a.ArchivingHash = 0
}

func (a *ArchivableImpl) Archived() bool {
	return a.ArchivingTimestamp != 0
}

func (a *ArchivableImpl) ArchiveMarks() (timeStamp UnixTime, archivingHash int64) {
	return a.ArchivingTimestamp, a.ArchivingHash
}
