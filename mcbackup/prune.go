package mcbackup

import (
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"github.com/spritsail/mcbackup/backup"
	"github.com/spritsail/mcbackup/config"
)

type PruneGroup struct {
	name    string
	reason  backup.Reason
	count   uint
	subTime func(time.Time, int) time.Time
}

func (mb *mcbackup) Prune(from time.Time) error {
	log := logrus.WithField("prefix", "prune")

	backups, err := mb.prov.List()
	if err != nil {
		return err
	}

	// Nothing to prune
	if len(backups) < 1 {
		log.Info("no backups to prune")
		return nil
	}

	// Ensure the backups are in a sorted order
	sort.Sort(backups)

	keep, remain, err := splitPrune(backups, mb.opts.Prune)

	if len(keep) > 0 {
		log.Infof("keeping %d backups", len(keep))
		for _, bkup := range keep {
			log.Tracef("  %s (%s)", bkup.Name(), bkup.Reason().String())
		}
		log.Tracef("keep %d + remain %d = all %d (%t)",
			len(keep), len(remain), len(backups),
			len(keep)+len(remain) == len(backups))
	}

	var spaceKeep uint64
	var sizeKeep uint64
	for _, bkup := range keep {
		// Calculate how much space we're using with the backups we're keeping
		sizeOnDisk, err := bkup.SpaceUsed()
		if err != nil {
			log.WithError(err).Warn("backup SpaceUsed failed")
		} else {
			spaceKeep += sizeOnDisk
		}

		realSize, err := bkup.Size()
		if err != nil {
			log.WithError(err).Warn("backup SpaceUsed failed")
		} else {
			sizeKeep += realSize
		}
	}
	log.Infof("%s used by %d backups (%s total size)", humanize.Bytes(spaceKeep), len(keep), humanize.Bytes(sizeKeep))

	if mb.opts.DryRun {
		log.Infof("actual prune would remove %d backups", len(remain))
	} else {
		log.Infof("removing %d backups", len(remain))
	}

	for _, bkup := range remain {
		log.Debugf("  %s (%s)", bkup.Name(), bkup.Reason().String())
	}

	// Quick, return before we actually delete anything
	if mb.opts.DryRun {
		return nil
	}

	var failed uint
	var removed uint
	var spaceSaved uint64
	var sizeSaved uint64
	for _, bkup := range remain {
		// Calculate how much space we're saving
		sizeOnDisk, err := bkup.SpaceUsed()
		if err != nil {
			log.WithError(err).Warn("backup.SpaceUsed() failed")
			sizeOnDisk = 0
		}
		realSize, err := bkup.Size()
		if err != nil {
			log.WithError(err).Warn("backup.Size() failed")
			realSize = 0
		}
		log.Tracef("deleting backup %s (%s/%s)", bkup.Name(), humanize.Bytes(sizeOnDisk), humanize.Bytes(realSize))

		if err = bkup.Delete(); err != nil {
			log.WithError(err).
				Warnf("failed to delete backup %s", bkup.Name())
			failed++
		} else {
			spaceSaved += sizeOnDisk
			sizeSaved += realSize
			removed++
		}
	}
	if failed > 0 {
		log.Errorf("failed to delete %d backups", failed)
	}

	log.Infof("%s saved in total with %d pruned backups (%s real size)", humanize.Bytes(spaceSaved),
		removed, humanize.Bytes(sizeSaved))

	return nil
}

func defaultPruneGroups(opts config.Prune) []PruneGroup {

	return []PruneGroup{
		{
			name:    "hour",
			reason:  backup.Hourly,
			count:   opts.KeepHourly,
			subTime: func(t time.Time, n int) time.Time { return t.Add(-time.Hour * time.Duration(n)) },
		},
		{
			name:    "day",
			reason:  backup.Daily,
			count:   opts.KeepDaily,
			subTime: func(t time.Time, n int) time.Time { return t.AddDate(0, 0, -n) },
		},
		{
			name:    "week",
			reason:  backup.Weekly,
			count:   opts.KeepWeekly,
			subTime: func(t time.Time, n int) time.Time { return t.AddDate(0, 0, -(n * 7)) },
		},
		{
			name:    "month",
			reason:  backup.Monthly,
			count:   opts.KeepMonthly,
			subTime: func(t time.Time, n int) time.Time { return t.AddDate(0, -n, 0) },
		},
		{
			name:    "year",
			reason:  backup.Yearly,
			count:   opts.KeepYearly,
			subTime: func(t time.Time, n int) time.Time { return t.AddDate(-n, 0, 0) },
		},
	}
}

// splitPrune separates a list of backups into two groups: keep and delete
func splitPrune(bs backup.Backups, opts config.Prune) (keep backup.Backups, remain backup.Backups, err error) {
	if len(bs) < 1 {
		return
	}

	log := logrus.WithField("prefix", "prune")

	// All backups we want to keep
	var keepMap = make(map[time.Time]backup.Backup, len(bs))

	// Always prune from the most recent backup
	// This prevents the situation of all backups being
	// deleted after starting the program after a long period of time
	var from = bs[len(bs)-1].When()

	var oldest = bs[0]
	log.Tracef("oldest: %s", oldest.When().Format(time.RFC3339))
	log.Tracef("now:    %s", from.Format(time.RFC3339))

	// start is the earlier of the two dates (beginning of the range)
	// end is the most recent of the two dates
	// end should be treated inclusively to not miss the most recent backup
	keepStart := from.Add(-opts.KeepFor)
	keepEnd := from

	for _, bkup := range bs {
		when := bkup.When()
		// Because we only check one group here, anything after `keepEnd' we
		// also want to keep as it is newer than "now" (although it should never
		// happen because "now" is the newest backup)
		if !when.Before(keepStart) {
			// Only insert the value if it's not already
			if _, ok := keepMap[when]; !ok {
				bkup.AddReason(backup.Recent)
				keepMap[bkup.When()] = bkup
			}
		} else {
			remain = append(remain, bkup)
		}
	}
	log.Debugf("keeping %d %s backups (%s)", len(keepMap),
		strings.ToLower(backup.Recent.String()), opts.KeepFor)

	// For each prune group
	groups := defaultPruneGroups(opts)
	for _, group := range groups {
		var numKept uint
		var inRange backup.Backups

		// Ensure toCheck is a shallow-copied slice of pointers
		var toCheck = make(backup.Backups, len(bs))
		copy(toCheck, bs)

		keepStart = group.subTime(from, 1)
		keepEnd = from

		numKept = 0

		// For each time period within the prune group (e.g. 1hr)
		for len(toCheck) > 0 && !keepEnd.Before(oldest.When()) && numKept < group.count {

			// Check each backup against each time period
			for _, bkup := range toCheck {
				if bkup == nil {
					continue
				}
				// Test if backup is within required range
				when := bkup.When()
				if when.After(keepStart) && !when.After(keepEnd) {
					inRange = append(inRange, bkup)
				}
			}

			// Choose the latest backup to keepMap from the time slot
			if len(inRange) > 0 {
				latest := inRange[len(inRange)-1]
				log.Tracef("latest between %s and %s is %s", keepStart.Format(time.RFC3339),
					keepEnd.Format(time.RFC3339), latest.Name())

				if latest.Reason()&group.reason != 0 {
					log.Warnf("adding keep entry for same reason. overlapping groups?  %s (%s)",
						latest.Name(), latest.Reason().String())
				} else {
					numKept++
				}
				latest.AddReason(group.reason)
				keepMap[latest.When()] = latest

				// Remove latest from remaining backups
				for idx, e := range remain {
					// Find the index of 'latest' and remove it from 'remaining'
					if e == latest {
						remain = append(remain[:idx], remain[idx+1:]...)
						break
					}
				}
			} else {
				log.Tracef("no backups between %s and %s", keepStart.Format(time.RFC3339),
					keepEnd.Format(time.RFC3339))
			}

			// Shift the time intervals down
			keepStart, keepEnd = group.subTime(keepStart, 1), keepStart

			// Empty inRange for the next range
			inRange = nil
		}

		log.Debugf("keeping %d %s backups for %d %ss", numKept,
			strings.ToLower(group.reason.String()), group.count, group.name)
	}

	// Retrieve backups to keepMap from map and sort them
	keep = make(backup.Backups, len(keepMap))
	var i uint
	for _, val := range keepMap {
		keep[i] = val
		i++
	}
	sort.Sort(keep)
	sort.Sort(remain)

	return
}
