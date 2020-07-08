package pool

import (
	"sync"
	"time"
)

const (
	R_STATUS_INITIALIZING = iota
	R_STATUS_STANDBY
	R_STATUS_RUNNING
	R_STATUS_FINISHED
	R_STATUS_SHUTING
	R_STATUS_DOWN
)

type routine struct {
	status     int
	statusLock *sync.Mutex
	signalCh   chan *signal
	reportCh   chan<- int
}

func newRoutine(expire time.Duration, reportCh chan<- int, taskCh <-chan *Runnable) *routine {
	r := &routine{
		status:     R_STATUS_INITIALIZING,
		statusLock: &sync.Mutex{},
		signalCh:   nil,
		reportCh:   reportCh,
	}

	r.up(expire, taskCh)

	return r
}

func (r *routine) up(expire time.Duration, taskCh <-chan *Runnable) bool {
	r.statusLock.Lock()
	defer r.statusLock.Unlock()

	status := r.getStatus()
	if !(status == R_STATUS_DOWN || status == R_STATUS_INITIALIZING) {
		return false
	}

	signalCh := make(chan *signal, 1)
	timeoutCh := time.After(expire)
	r.signalCh = signalCh

	go r._run(signalCh, timeoutCh, taskCh)

	return true
}

func (r *routine) shutdown() bool {
	r.statusLock.Lock()
	defer r.statusLock.Unlock()

	if status := r.getStatus(); status == R_STATUS_DOWN || status == R_STATUS_SHUTING {
		return false
	}

	r.signalCh <- newExitSignal()
	close(r.signalCh)
	r.signalCh = nil

	return true
}

func (r *routine) _run(signalCh <-chan *signal, timeoutCh <-chan time.Time, taskCh <-chan *Runnable) {
	defer r._setStatus(R_STATUS_DOWN)

	r._setStatus(R_STATUS_STANDBY)
	for {
		select {
		case <-timeoutCh:
			r._setStatus(R_STATUS_SHUTING)
			return
		case <-signalCh:
			r._setStatus(R_STATUS_SHUTING)
			return
		default:

			select {
			case <-timeoutCh:
				r._setStatus(R_STATUS_SHUTING)
				return
			case <-signalCh:
				r._setStatus(R_STATUS_SHUTING)
				return
			case t := <-taskCh:
				r._setStatus(R_STATUS_RUNNING)
				(*t).Run()
				r._setStatus(R_STATUS_FINISHED)
				r._setStatus(R_STATUS_STANDBY)
			}
		}
	}
}

func (r *routine) _setStatus(status int) {
	r.status = status
	r.reportCh <- status
}

func (r *routine) getStatus() int {
	return r.status
}

const (
	SIG_EXIT = iota
)

func newExitSignal() *signal {
	return newSignal(SIG_EXIT)
}

func newSignal(signalId int) *signal {
	s := &signal{}
	s.set(signalId)

	return s
}

type signal struct {
	signalId int
}

func (s *signal) get() int {
	return s.signalId
}

func (s *signal) set(signalId int) {
	s.signalId = signalId
}
