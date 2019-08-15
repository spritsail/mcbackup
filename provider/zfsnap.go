package provider

import (
	"strconv"
	"time"

	zfs "github.com/bicomsystems/go-libzfs"
	"github.com/spritsail/mcbackup/backup"
)

type zfsSnapshot struct {
	dataset string

	name   string
	when   time.Time
	reason backup.Reason
}

func (zs *zfsSnapshot) Name() string {
	return zs.name
}

func (zs *zfsSnapshot) When() time.Time {
	return zs.when
}

func (zs *zfsSnapshot) Delete() error {
	ds, err := zfs.DatasetOpen(zs.dataset)
	defer ds.Close()
	if err != nil {
		return err
	}
	return ds.Destroy(true)
}

func (zs *zfsSnapshot) Size() (uint64, error) {
	ds, err := zfs.DatasetOpen(zs.dataset)
	defer ds.Close()
	if err != nil {
		return 0, err
	}
	prop := ds.Properties[zfs.DatasetPropReferenced].Value
	return strconv.ParseUint(prop, 10, 64)
}

func (zs *zfsSnapshot) SpaceUsed() (uint64, error) {
	ds, err := zfs.DatasetOpen(zs.dataset)
	defer ds.Close()
	if err != nil {
		return 0, err
	}
	prop := ds.Properties[zfs.DatasetPropUsed].Value
	return strconv.ParseUint(prop, 10, 64)
}

func (zs *zfsSnapshot) Reason() backup.Reason {
	return zs.reason
}

func (zs *zfsSnapshot) AddReason(r backup.Reason) {
	zs.reason |= r
}

func (zs *zfsSnapshot) SetReason(r backup.Reason) {
	zs.reason = r
}

var _ backup.Backup = &zfsSnapshot{}
