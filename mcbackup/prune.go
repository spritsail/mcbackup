package mcbackup

import (
	"sort"
	"strings"
	"time"

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
	for _, bkup := range remain {
		log.Tracef("deleting backup %s", bkup.Name())
		if err = bkup.Delete(); err != nil {
			log.Warnf("failed to delete backup %s", bkup.Name())
			failed++
		}
	}
	log.Errorf("failed to delete %d backups", failed)

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
	log.Tracef("oldest: %s", oldest.When())
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
		var iterations, numKept uint
		var inRange backup.Backups

		// Ensure toCheck is a shallow-copied slice of pointers
		var toCheck = make(backup.Backups, len(bs))
		copy(toCheck, bs)
		log.Tracef("from %s, to now", keepEnd.Format(time.RFC3339))

		keepStart = group.subTime(from, 1)
		keepEnd = from

		iterations, numKept = 0, 0
		// For each time period within the prune group (e.g. 1hr)
		for len(toCheck) > 0 && !keepEnd.Before(oldest.When()) && iterations < group.count {

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
				if latest.Reason()&group.reason != 0 {
					log.Warnf("adding keep entry for same reason. overlapping groups?  %s (%s)",
						latest.Name(), latest.Reason().String())
				}
				latest.AddReason(group.reason)
				keepMap[latest.When()] = latest
				numKept++

				// Remove latest from remaining backups
				for idx, e := range remain {
					// Find the index of 'latest' and remove it from 'remaining'
					if e == latest {
						remain = append(remain[:idx], remain[idx+1:]...)
						break
					}
				}
			}

			// Shift the time intervals down
			keepStart, keepEnd = group.subTime(keepStart, 1), keepStart

			// Empty inRange for the next range
			inRange = nil

			iterations++
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
