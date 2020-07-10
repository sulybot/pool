package pool

import "time"

const (
	routineStatusDown    = 0
	routineStatusStandby = 1
	routineStatusWorking = 2
)

type routine uint8

func (r *routine) run(idleQueue chan<- chan Runnable, idleTimeout int64) {
	if 0 == idleTimeout {
		r.foreverRun(idleQueue)
	} else {
		r.expireRun(idleQueue, idleTimeout)
	}
}

func (r *routine) foreverRun(idleQueue chan<- chan Runnable) {
	defer r.setStatus(routineStatusDown)

	taskCh := make(chan Runnable)
	defer close(taskCh)

	for {
		r.setStatus(routineStatusStandby)

		select {
		case idleQueue <- taskCh:
			select {
			case task, ok := <-taskCh:
				if !ok {
					return
				}

				r.setStatus(routineStatusWorking)
				task.Run()
			}
		}
	}
}

func (r *routine) expireRun(idleQueue chan<- chan Runnable, idleTimeout int64) {
	defer r.setStatus(routineStatusDown)

	taskCh := make(chan Runnable)
	defer close(taskCh)

	idleDuration := time.Duration(idleTimeout) * time.Second
	timer := time.NewTimer(idleDuration)
	defer timer.Stop()

	for {
		r.setStatus(routineStatusStandby)

		select {
		case <-timer.C:
			return

		case idleQueue <- taskCh:
			select {
			case task, ok := <-taskCh:
				if !ok {
					return
				}

				r.setStatus(routineStatusWorking)
				task.Run()

				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(idleDuration)
			}
		}
	}
}

func (r *routine) setStatus(status uint8) {
	*r = routine(status)
}

func (r *routine) status() uint8 {
	return uint8(*r)
}

type Runnable interface {
	Run()
}
