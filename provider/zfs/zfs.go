package zfs

import (
	strftime "github.com/cactus/gostrftime"
	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"
	"github.com/lorenz/go-libzfs"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/provider"
	"strconv"
	"time"
)

type ZfsProvider struct {
	Dataset    string `long:"zfs-dataset" description:"Dataset/volume name" env:"ZFS_DATASET" required:"true"`
	Recursive  bool   `long:"zfs-recursive" description:"Should snapshots be recursive" env:"ZFS_SNAPSHOT_RECURSE"`
	DateFormat string `long:"zfs-date-format" description:"Format for snapshot names" env:"ZFS_DATE_FORMAT" default:"%F-%H:%M"`
}

func New(args []string) (p provider.Provider, remain []string, err error) {
	var zfsOpts ZfsProvider

	parser := flags.NewParser(&zfsOpts, flags.IgnoreUnknown)
	remain, err = parser.ParseArgs(args)
	if err != nil {
		return
	}

	d, err := zfs.DatasetOpen(zfsOpts.Dataset)
	defer d.Close()
	if err != nil {
		return
	}

	p = &zfsOpts
	return
}

func (zp *ZfsProvider) TakeBackup() error {
	log := logrus.WithField("prefix", "zfs")

	log.Info("taking zfs snapshot")

	// Take the snapshot and return the error if any
	snapName := zp.Dataset + "@" + zp.genSnapshotName()
	props := make(map[zfs.Prop]zfs.Property)
	snap, err := zfs.DatasetSnapshot(snapName, zp.Recursive, props)
	if err != nil {
		return err
	}
	defer snap.Close()

	// Parse reference size and print a pretty message
	refSizeStr := snap.Properties[zfs.DatasetPropReferenced].Value
	refSize, _ := strconv.ParseUint(refSizeStr, 10, 64)
	log.Infof("snapshot %s created, %s refer size", snapName,
		humanize.Bytes(refSize))

	return nil
}

func (zp *ZfsProvider) genSnapshotName() string {
	return strftime.Format(zp.DateFormat, time.Now())
}
