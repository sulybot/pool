package pool

import "time"

const (
	ROUTINE_STATUS_DOWN    = 0
	ROUTINE_STATUS_STANDBY = 1
	ROUTINE_STATUS_WORKING = 2
)

type routine struct {
	status int
	taskCh chan Runnable
}

func (r *routine) run(routineStandByCh chan<- chan Runnable, ttl uint) {
	r.status = ROUTINE_STATUS_STANDBY
	defer func() {
		r.status = ROUTINE_STATUS_DOWN
	}()

	if nil == r.taskCh {
		r.taskCh = make(chan Runnable)
		defer close(r.taskCh)
	}

	var (
		longLive  bool = 0 == ttl
		timer     *time.Timer
		timeoutCh <-chan time.Time
	)

	if !longLive {
		timer = time.NewTimer(time.Duration(ttl))
		timeoutCh = timer.C
	} else {
		timeoutCh = make(<-chan time.Time, 0)
	}

	for {
		r.status = ROUTINE_STATUS_STANDBY

		select {
		case routineStandByCh <- r.taskCh:
			select {
			case task, ok := <-r.taskCh:
				r.status = ROUTINE_STATUS_WORKING
				if !longLive && !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}

				if !ok {
					return
				}

				task.Run()
				if !longLive {
					timer.Reset(time.Duration(ttl))
				}
			}

		case <-timeoutCh:
			return
		}
	}
}

type Runnable interface {
	Run()
}
