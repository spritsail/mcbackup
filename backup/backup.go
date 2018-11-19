package backup

import (
	"sort"
	"strings"
	"time"
)

type Reason uint

const (
	Unknown Reason = 1 << 0
	Recent  Reason = 1 << 1
	Hourly  Reason = 1 << 2
	Daily   Reason = 1 << 3
	Weekly  Reason = 1 << 4
	Monthly Reason = 1 << 5
	Yearly  Reason = 1 << 6
)

func (reason Reason) String() string {
	var ss []string
	if reason&Recent != 0 {
		ss = append(ss, "Recent")
	}
	if reason&Hourly != 0 {
		ss = append(ss, "Hourly")
	}
	if reason&Daily != 0 {
		ss = append(ss, "Daily")
	}
	if reason&Weekly != 0 {
		ss = append(ss, "Weekly")
	}
	if reason&Monthly != 0 {
		ss = append(ss, "Monthly")
	}
	if reason&Yearly != 0 {
		ss = append(ss, "Yearly")
	}
	if len(ss) < 1 {
		return "Unknown"
	}
	return strings.Join(ss, "|")
}

type Backup interface {
	Name() string
	When() time.Time
	Remove() error
	Size() (uint64, error)
	SpaceUsed() (uint64, error)

	Reason() Reason
	AddReason(Reason)
	SetReason(Reason)
}

type Backups []Backup

func (bs Backups) Len() int {
	return len(bs)
}
func (bs Backups) Less(i, j int) bool {
	return bs[i].When().Before(bs[j].When())
}

func (bs Backups) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}

// Compile-time implementation checks
var _ sort.Interface = Backups{}
