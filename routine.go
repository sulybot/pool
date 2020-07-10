package pool

import "time"

const (
	routineStatusDown    = 0
	routineStatusStandby = 1
	routineStatusWorking = 2
)

// The routine type represents a goroutine.
// Its value is a uint8, it will be the one of the const routineStatus.
// A routine instance is bind with a goroutine (see method routine.run),
// and the run func will change the routine value.
type routine uint8

// The run method works as the entrance of the goroutine.

// Arg idleQueue is an unbuffered [chan<- chan Runnable] type.
// after run started, it will push a taskCh[chan Runnable] into idleQueue
// to notify the pool that the routine is ready for handling task.
// taskCh is also unbuffered.
// Once the he pool selected a taskCh, the routine will waiting for the task from taskCh.
// A task is the type which is implements Runnable interface.

// Arg idleTimeout represents the time to wait of the routine,
// if idleTimeout is 0, the routine will never timeout.
func (r *routine) run(idleQueue chan<- chan Runnable, idleTimeout int64) {
	if 0 == idleTimeout {
		r.foreverRun(idleQueue)
	} else {
		r.expireRun(idleQueue, idleTimeout)
	}
}

// foreverRun is the routine without timeout
func (r *routine) foreverRun(idleQueue chan<- chan Runnable) {
	defer r.setStatus(routineStatusDown)

	taskCh := make(chan Runnable)
	defer close(taskCh)

	for {
		r.setStatus(routineStatusStandby)

		select {
		// An idleQueue is an unbuffered chan,
		// it means the pushing of taskCh will be blocked.
		// Once the taskCh push into the idleQueue,
		// the routine is being selected to handle task.
		case idleQueue <- taskCh:
			select {
			case task, ok := <-taskCh:
				if !ok {
					// If the taskCh is selected but closed,
					// it means the pool is shuting down the routine manually.
					return
				}

				r.setStatus(routineStatusWorking)
				task.Run()
			}
		}
	}
}

//expireRun is the routine with timeout
func (r *routine) expireRun(idleQueue chan<- chan Runnable, idleTimeout int64) {
	defer r.setStatus(routineStatusDown)

	taskCh := make(chan Runnable)
	idleDuration := time.Duration(idleTimeout) * time.Second
	timer := time.NewTimer(idleDuration) // timer is created for timeout
	defer timer.Stop()

	for {
		r.setStatus(routineStatusStandby)

		select {
		// if the timer timeout, we will stop the routine
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

				// timer may timeout during the task handling,
				// no matter what, we reset the timer, let the routine goes on.
				if !timer.Stop() {
					// If the timer push the timeout time,
					// it must be popped before the next loop to avoid the terminated of routine.
					<-timer.C
				}
				timer.Reset(idleDuration)
			}
		}
	}
}

// setStatus is called in run,
// it also called before the pool forking a routine.
func (r *routine) setStatus(status uint8) {
	*r = routine(status)
}

// status reports the status of the routine.
func (r *routine) status() uint8 {
	return uint8(*r)
}

// Runnable represents a task which will call in routine.
// It is the contract of routine.
type Runnable interface {
	Run()
}
