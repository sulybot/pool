package pool

import (
	"sync"
	"time"
)

const (
	R_STATUS_INITIALIZING = iota
	R_STATUS_STANDBY
	R_STATUS_RUNNING
	R_STATUS_SHUTING
	R_STATUS_DOWN
)

type routine struct {
	status     int
	signalCh   chan *signal
	statusLock *sync.Mutex
}

func newRoutine(expire time.Duration) *routine {
	r := &routine{
		status:     R_STATUS_INITIALIZING,
		signalCh:   nil,
		statusLock: &sync.Mutex{},
	}

	r.up(expire)

	return r
}

func (r *routine) up(expire time.Duration) bool {
	r.statusLock.Lock()
	defer r.statusLock.Unlock()

	if r.getStatus() != R_STATUS_DOWN {
		return false
	}

	signalCh := make(chan *signal, 1)
	timeoutCh := time.After(expire)
	r.signalCh = signalCh

	go r._run(signalCh, timeoutCh)

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

func (r *routine) _run(signalCh <-chan *signal, timeoutCh <-chan time.Time) {
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
				//todo: task case
			}
		}
	}
}

func (r *routine) _setStatus(status int) {
	r.status = status
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
