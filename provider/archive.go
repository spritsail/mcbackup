package provider

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type ArchiveProvider struct {
	SourceDirectory string `short:"s" long:"source-dir" description:"Minecraft server directory to backup" env:"SOURCE_DIRECTORY" required:"true"`
	BackupDirectory string `short:"b" long:"backup-dir" description:"Directory to save backup archives" env:"BACKUP_DIRECTORY" required:"true"`
}

func (opts *ArchiveProvider) InitArchive() (err error) {
	err = checkDirectory(opts.BackupDirectory, "backup")
	if err != nil {
		return
	}

	return checkDirectory(opts.SourceDirectory, "source")
}

func checkDirectory(path string, typ string) (err error) {
	log := logrus.WithField("prefix", "archive")

	// perform various BackupDirectory checks
	dir, err := os.Stat(path)
	if err != nil {
		// Attempt to create the directory
		err = os.Mkdir(path, 0755)
		if err != nil {
			log.WithField("dir", path).
				WithError(err).
				Warnf("failed to create %s directory", typ)
			return
		}

		// Check again now we've created it
		dir, err = os.Stat(path)
		if err != nil {
			log.WithField("dir", path).
				Warnf("%s directory is inaccessible", typ)
			return
		}
	}
	if !dir.IsDir() {
		return fmt.Errorf("path `%s' is not a directory", path)
	}
	if unix.Access(path, unix.W_OK) != nil {
		err = os.ErrPermission
		log.WithField("dir", path).
			WithError(err).
			Warnf("%s directory is not writable", typ)
		return
	}

	return
}
