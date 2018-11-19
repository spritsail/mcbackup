package provider

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"
	"github.com/lorenz/go-libzfs"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/backup"
	"github.com/spritsail/mcbackup/config"
)

type ZfsProvider struct {
	opts      *config.Options
	Dataset   string `long:"zfs-dataset" description:"Dataset/volume name" env:"ZFS_DATASET" required:"true"`
	Recursive bool   `long:"zfs-recursive" description:"Should snapshots be recursive" env:"ZFS_SNAPSHOT_RECURSE"`
}

func NewZFS(args []string, opts *config.Options) (p Provider, remain []string, err error) {
	var zfsOpts ZfsProvider
	zfsOpts.opts = opts

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

func (zp *ZfsProvider) Create(name string, when time.Time) (backup.Backup, error) {
	log := logrus.WithField("prefix", "zfs")

	log.Info("taking zfs snapshot")

	// Take the snapshot and return the error if any
	snapName := zp.Dataset + "@" + name
	props := make(map[zfs.Prop]zfs.Property)
	snap, err := zfs.DatasetSnapshot(snapName, zp.Recursive, props)
	if err != nil {
		return nil, err
	}
	defer snap.Close()

	// Parse reference size and print a pretty message
	refSizeStr := snap.Properties[zfs.DatasetPropReferenced].Value
	refSize, _ := strconv.ParseUint(refSizeStr, 10, 64)
	log.Infof("snapshot %s created, %s refer size", snapName,
		humanize.Bytes(refSize))

	bkup := &zfsSnapshot{
		dataset: snapName,
		name:    name,
		when:    when,
		reason:  backup.Unknown,
	}

	return bkup, nil
}

func (zp *ZfsProvider) List() (bs backup.Backups, err error) {
	ds, err := zfs.DatasetOpen(zp.Dataset)
	defer ds.Close()
	if err != nil {
		return
	}

	for _, child := range ds.Children {
		name := child.Properties[zfs.DatasetPropName].Value

		if !(child.Type == zfs.DatasetTypeSnapshot ||
			strings.Contains(name, "@")) {
			continue
		}

		parts := strings.Split(name, "@")
		dsetName, snapName := parts[0], parts[1]

		if dsetName != zp.Dataset {
			logrus.WithField("prefix", "zfs").
				Warn("dataset snapshot with different name to parent")
			continue
		}

		// Ensure the backup is valid, then create a
		if zp.opts.IsMcbackup(snapName) {
			when, err := zp.opts.ParseBackupName(snapName)
			if err != nil {
				return nil, err
			}

			bs = append(bs, &zfsSnapshot{
				dataset: name,
				name:    snapName,
				when:    when,
				reason:  backup.Unknown,
			})
		}
	}
	return bs, nil
}

func (zp *ZfsProvider) Remove(bkup backup.Backup) error {
	// Ensure backup is a zfs snapshot
	switch snap := bkup.(type) {
	case *zfsSnapshot:
		ds, err := zfs.DatasetOpen(snap.dataset)
		defer ds.Close()
		if err != nil {
			return err
		}
		return ds.Destroy(true)
	default:
		return fmt.Errorf("backup is not a zfs snapshot")
	}
}

var _ Provider = &ZfsProvider{}
