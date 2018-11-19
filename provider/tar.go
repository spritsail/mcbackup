package provider

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mholt/archiver"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/backup"
	"github.com/spritsail/mcbackup/config"
)

var log *logrus.Entry

func init() {
	log = logrus.WithField("prefix", "tar")
}

type TarArchiver interface {
	archiver.Archiver
	fmt.Stringer
}

type TarProvider struct {
	ArchiveProvider
	opts      *config.Options
	tar       TarArchiver
	Algo      string `short:"c" long:"tar-compression" description:"compression algorithm used for the tar archive" env:"TAR_COMPRESSION" default:"gzip"`
	Extension string `long:"tar-extension" description:"file extension used for backup archives"`
	Level     int    `long:"compression-level" description:"level of the compression (algorithm dependent)" env:"COMPRESSION_LEVEL"`
}

func NewTar(args []string, opts *config.Options) (p Provider, remain []string, err error) {
	var tarOpts TarProvider
	tarOpts.opts = opts

	parser := flags.NewParser(&tarOpts, flags.IgnoreUnknown)
	remain, err = parser.ParseArgs(args)
	if err != nil {
		return
	}

	// Attempt to initialise archive-global options
	err = tarOpts.InitArchive()

	// TODO: Validate the CompressionLevel now instead of later
	switch tarOpts.Algo {
	case "gz":
		fallthrough
	case "gzip":
		tarOpts.tar = &archiver.TarGz{
			Tar:              archiver.DefaultTar,
			CompressionLevel: tarOpts.Level,
		}
	case "bz2":
		fallthrough
	case "bzip2":
		tarOpts.tar = &archiver.TarBz2{
			Tar:              archiver.DefaultTar,
			CompressionLevel: tarOpts.Level,
		}
	case "lz4":
		tarOpts.tar = &archiver.TarLz4{
			Tar:              archiver.DefaultTar,
			CompressionLevel: tarOpts.Level,
		}
	case "xz":
		tarOpts.tar = archiver.DefaultTarXz
	default:
		err = fmt.Errorf("unknown compression algorithm '%s'", tarOpts.Algo)
		return
	}

	// Default and sanitise file extension
	if tarOpts.Extension == "" {
		tarOpts.Extension = tarOpts.tar.String()
		log.WithField("extension", tarOpts.Extension).
			Debugf("Using default file extension")
	}

	// Validate the file extension against the compression algo
	err = tarOpts.tar.CheckExt("testfilename." + tarOpts.Extension)
	if err != nil {
		return
	}

	p = &tarOpts
	return
}

func (tp *TarProvider) Create(name string, when time.Time) (backup.Backup, error) {
	filename := name + "." + tp.Extension
	filepath := path.Join(tp.BackupDirectory, filename)
	log.WithField("filename", filename).Debugf("creating tar backup")

	// Create the backup
	err := tp.tar.Archive([]string{tp.SourceDirectory}, filepath)
	if err != nil {
		return nil, err
	}

	bkup := &ArchiveBackup{
		path:   filepath,
		name:   name,
		when:   when,
		reason: backup.Unknown,
	}

	return bkup, err
}

func (tp *TarProvider) List() (backup.Backups, error) {
	infos, err := ioutil.ReadDir(tp.ArchiveProvider.BackupDirectory)
	if err != nil {
		return nil, err
	}

	var bkups backup.Backups
	for _, info := range infos {
		if !tp.opts.IsMcbackup(info.Name()) {
			continue
		}

		when, err := tp.opts.ParseBackupName(info.Name())
		if err != nil {
			return nil, err
		}
		// Recreate the backup name from the parsed value
		// This gets around the issue of trying to remove the file extension
		backupName, err := tp.opts.GenBackupName(when)
		if err != nil {
			return nil, err
		}
		archiveBackup := &ArchiveBackup{
			path:   path.Join(tp.ArchiveProvider.BackupDirectory, info.Name()),
			name:   backupName,
			when:   when,
			reason: backup.Unknown,
		}
		bkups = append(bkups, archiveBackup)
	}

	return bkups, nil
}

var _ Provider = &TarProvider{}
