package provider

import (
	"os"
	"time"

	"github.com/spritsail/mcbackup/backup"
)

type ArchiveBackup struct {
	path   string
	name   string
	when   time.Time
	reason backup.Reason
}

func (ab *ArchiveBackup) Name() string {
	return ab.name
}

func (ab *ArchiveBackup) When() time.Time {
	return ab.when
}

func (ab *ArchiveBackup) Delete() error {
	return os.Remove(ab.path)
}

func (ab *ArchiveBackup) Size() (uint64, error) {
	file, err := os.Stat(ab.path)
	if err != nil {
		return 0, err
	}
	return uint64(file.Size()), nil
}

func (ab *ArchiveBackup) SpaceUsed() (uint64, error) {
	// TODO: Find size of file on disk
	return ab.Size()
}

func (ab *ArchiveBackup) Reason() backup.Reason {
	return ab.reason
}

func (ab *ArchiveBackup) AddReason(r backup.Reason) {
	ab.reason |= r
}

func (ab *ArchiveBackup) SetReason(r backup.Reason) {
	ab.reason = r
}

var _ backup.Backup = &ArchiveBackup{}
