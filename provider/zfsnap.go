package provider

import (
	"time"

	"github.com/spritsail/mcbackup/backup"
)

type zfsSnapshot struct {
	data    backup.Data
	dataset string
}

func (zs *zfsSnapshot) Name() string {
	return zs.data.Name()
}

func (zs *zfsSnapshot) When() time.Time {
	return zs.data.When()
}

func (zs *zfsSnapshot) Reason() backup.Reason {
	return zs.data.Reason()
}

func (zs *zfsSnapshot) AddReason(r backup.Reason) {
	zs.data.AddReason(r)
}

func (zs *zfsSnapshot) SetReason(r backup.Reason) {
	zs.data.SetReason(r)
}

var _ backup.Backup = &zfsSnapshot{}
