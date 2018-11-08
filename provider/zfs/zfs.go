package zfs

import (
	strftime "github.com/cactus/gostrftime"
	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"
	"github.com/mistifyio/go-zfs"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/provider"
	"time"
)

type ZfsProvider struct {
	Dataset    string `long:"zfs-dataset" description:"Dataset/volume name" env:"ZFS_DATASET" required:"true"`
	Recursive  bool   `long:"zfs-recursive" description:"Should snapshots be recursive" env:"ZFS_SNAPSHOT_RECURSE"`
	DateFormat string `long:"zfs-date-format" description:"Format for snapshot names" env:"ZFS_DATE_FORMAT" default:"%F-%H:%M:%S"`
}

func New(args []string) (p provider.Provider, remain []string, err error) {
	var zfsOpts ZfsProvider

	parser := flags.NewParser(&zfsOpts, flags.IgnoreUnknown)
	remain, err = parser.ParseArgs(args)
	if err != nil {
		return
	}

	_, err = zfs.GetDataset(zfsOpts.Dataset)
	if err != nil {
		return
	}

	p = &zfsOpts
	return
}

func (zp *ZfsProvider) TakeBackup() error {
	log := logrus.WithField("prefix", "zfs")

	log.Debugf("finding zfs dataset %s", zp.Dataset)

	// Obtain a handle to the dataset
	// This occurs every time to ensure the dataset still exists
	dataset, err := zfs.GetDataset(zp.Dataset)
	if err != nil {
		return err
	}

	log.Info("taking zfs snapshot")

	// Take the snapshot and return the error if any
	snap, err := dataset.Snapshot(zp.genSnapshotName(), zp.Recursive)
	log.Infof("snapshot %s created, %s refer size", snap.Name,
		humanize.Bytes(snap.Referenced))

	return err
}

func (zp *ZfsProvider) genSnapshotName() string {
	return strftime.Format(zp.DateFormat, time.Now())
}
