package backup

import "time"

type Data struct {
	name   string
	when   time.Time
	reason Reason
}

func NewData(name string, when time.Time, reason Reason) Data {
	return Data{name, when, reason}
}

func (d *Data) Name() string {
	return d.name
}

func (d *Data) When() time.Time {
	return d.when
}

func (d *Data) Reason() Reason {
	return d.reason
}

func (d *Data) AddReason(r Reason) {
	d.reason |= r
}

func (d *Data) SetReason(r Reason) {
	d.reason = r
}

var _ Backup = &Data{}
