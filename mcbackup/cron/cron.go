package cron

import (
	"github.com/gorhill/cronexpr"
	"github.com/sirupsen/logrus"
	"time"
)

type task struct {
	Job        func() error         // function to run
	When       *cronexpr.Expression // when to run it
	Done       chan error           // channel, called when the job is complete. if channel is closed, the job has already finished
	ErrHandler func(error)          // optional function to handle errors, can be used to stop the timer
	running    bool                 // set false
	cancel     chan struct{}        // a channel to interrupt the sleeping loop
}

func Schedule(when string, what func() error) (*task, error) {

	whenExpr, err := cronexpr.Parse(when)
	if err != nil {
		return nil, err
	}

	return &task{
		Job:     what,
		When:    whenExpr,
		running: false,
	}, err
}

func (t *task) NextRun() time.Time {
	return t.When.Next(time.Now())
}

func (t *task) Run() {
	log := logrus.WithField("prefix", "cron")

	t.running = true
	t.Done = make(chan error, 1)
	t.cancel = make(chan struct{}, 1)
	var err error
	for t.running {

		// Wait until the next next starts
		now := time.Now()
		next := t.When.Next(now)
		waitfor := next.Sub(now)
		log.Debug("waiting until, ", next)
		select {
		case <-time.After(waitfor):
			break
		case <-t.cancel:
			close(t.Done)
			return
		}

		log.Debug("executing job")
		err = t.Job()

		if err != nil {
			log.WithError(err).
				Debug("job failed")

			if t.ErrHandler != nil {
				// Pass the error to the handler
				t.ErrHandler(err)
			}
		} else {
			log.Debug("job completed")
		}
	}

	// Send the error/nil to signify end of job
	t.Done <- err
	close(t.Done)
}

func (t *task) Cancel() {
	t.running = false
	// closing the channel should be enough to wake the loop
	close(t.cancel)
}
