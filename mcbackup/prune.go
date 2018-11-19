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

	keep, remain, err := splitPrune(backups, from, mb.opts.Prune)

	if len(keep) > 0 {
		log.Infof("keeping %d backups", len(keep))
		for _, bkup := range keep {
			log.Tracef("  %s (%s)", bkup.Name(), bkup.Reason().String())
		}
		log.Tracef("keep %d + remain %d = all %d (%t)",
			len(keep), len(remain), len(backups),
			len(keep)+len(remain) == len(backups))
	}

	// Quick, return before we actually delete anything
	if mb.opts.DryRun {
		return nil
	}

	log.Infof("removing %d backups", len(remain))
	for _, bkup := range remain {
		log.Tracef("  %s (%s)", bkup.Name(), bkup.Reason().String())
	}

	for _, bkup := range remain {
		if err = bkup.Delete(); err != nil {
			return err
		}
	}

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
func splitPrune(bs backup.Backups, from time.Time, opts config.Prune) (keep backup.Backups, remain backup.Backups, err error) {
	if len(bs) < 1 {
		return
	}

	log := logrus.WithField("prefix", "prune")

	// All backups we want to keep
	var keepMap = make(map[time.Time]backup.Backup, len(bs))

	var oldest = bs[0]
	log.Tracef("oldest: %+v", oldest.Name())
	log.Tracef("now %s", from.Format(time.RFC3339))

	keepStart := from.Add(-opts.KeepFor)
	keepEnd := from

	for _, bkup := range bs {
		when := bkup.When()
		if when.After(from) || when.Equal(from) ||
			(when.After(keepStart) && when.Before(keepEnd)) {
			// Only insert the value if it's not already
			if _, ok := keepMap[when]; !ok {
				bkup.AddReason(backup.Recent)
				keepMap[bkup.When()] = bkup
			}
		} else {
			remain = append(remain, bkup)
		}
	}
	log.Debugf("keeping %d %s backups", len(keepMap),
		strings.ToLower(backup.Recent.String()))

	// For each prune group
	groups := defaultPruneGroups(opts)
	for _, group := range groups {
		var iterations, numKept uint
		var inRange backup.Backups

		// Ensure toCheck is a shallow-copied slice of pointers
		var toCheck = make(backup.Backups, len(bs))
		copy(toCheck, bs)
		log.Tracef("from %s, to now", keepEnd.Format(time.RFC3339))

		keepStart = from
		keepEnd = group.subTime(from, 1)

		iterations, numKept = 0, 0
		// For each time period within the prune group (e.g. 1hr)
		for len(toCheck) > 0 && keepStart.After(oldest.When()) && iterations < group.count {

			// Check each backup against each time period
			for _, bkup := range toCheck {
				if bkup == nil {
					continue
				}
				// Test if backup is within required range
				when := bkup.When()
				if when.After(keepEnd) && when.Before(keepStart) {
					inRange = append(inRange, bkup)
				}
			}

			// Choose the latest backup to keepMap from the time slot
			if len(inRange) > 0 {
				latest := inRange[len(inRange)-1]
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
			}

			// Shift the time intervals down
			keepEnd, keepStart = group.subTime(keepEnd, 1), keepEnd

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
