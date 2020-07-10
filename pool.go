package pool

// Pool represents a goroutine pool.
type Pool struct {
	idleTimeout int64
	routinePool []*routine
	taskQueue   chan Runnable
	termSignal  chan bool
}

// New return a new pool.
// It accept two args, maxRoutine and taskQueueSize.
// maxRoutine represents how many goroutine can be forked in the pool.
// taskQueueSize represents the task queue size.
// When all of the goroutines are working and more task is start,
// the new task will cached in a taskQueue and waiting for the standby routine.
func New( /*maxRoutine, taskQueueSize int*/ maxAndSize ...int) *Pool {
	var maxRoutine, taskQueueSize int

inital:
	for k, v := range maxAndSize {
		switch k {
		case 0: //maxRoutine
			maxRoutine = v
		case 1: //queueSize
			taskQueueSize = v
		default:
			break inital
		}
	}

	p := &Pool{
		idleTimeout: 10,
		routinePool: make([]*routine, maxRoutine, maxRoutine),
		taskQueue:   make(chan Runnable, taskQueueSize),
	}

	// initialize the routine pool slice
	for i := 0; i < maxRoutine; i++ {
		p.routinePool[i] = new(routine)
	}

	go p.masterRoutine()
	return p
}

// SetIdleTimeout will set the idle timeout of the routine in the pool.
// SetIdleTimeout is ok to be called at anytime, but only affected to the newly forked routine.
func (p *Pool) SetIdleTimeout(idleTimeout int64) {
	p.idleTimeout = idleTimeout
}

// masterRoutine is the master routine of handling task.
// it only called once when creating a pool in pool.New.
func (p *Pool) masterRoutine() {
	idleQueue := make(chan chan Runnable)
	tryFork := make(chan bool, 1)
	defer close(idleQueue)
	defer close(tryFork)

	for {
		select {
		// Firstly, master routine try to pop a task from taskQueue.
		case task, ok := <-p.taskQueue:
			if !ok {
				// If taskQueue was closed, master routine will shutdown all the routines gracefully.
				// In the for loop, we remove all the down routines until the routine pool is empty.
				for p.removeDownRoutine(); 0 < len(p.routinePool); p.removeDownRoutine() {
					select {
					case taskCh := <-idleQueue:
						// close the taskCh, let the routine return
						// the routine status will be set to down
						close(taskCh)
					default:
					}
				}

				// if all the routines shuteddown successfully, send the finish signal.
				p.termSignal <- true
				return
			}

			// Secondly, master routine will try to pop a taskCh,
			// then push the task into the taskCh, let the routine handle it.
			tryFork <- true
		popRoutineLoop:
			for {
				select {
				case taskCh := <-idleQueue:
					taskCh <- task // If taskCh select successful, just push task into the routine.
					<-tryFork      // clear the tryFork for the next newly task
					break popRoutineLoop

				default:
					// If no standby taskCh, we try to fork a new one.
					// but fork should do only onetime
					for {
						select {
						// Fork for first time, the second time will block
						case <-tryFork:
							p.fork(idleQueue)

							// If new routine truly forked, just push the task into it
							// But if the pool is full of working routine,
							//there is nothing to do but wait.
						case taskCh := <-idleQueue:
							taskCh <- task

							if 0 < len(tryFork) {
								<-tryFork
							}
							break popRoutineLoop
						}
					}
				}
			}
		}
	}
}

// fork try to start a new routine,
// it travel through the routinePool and try to find a routine with status down.
func (p *Pool) fork(idleQueue chan<- chan Runnable) bool {
	for _, routine := range p.routinePool {
		if routineStatusDown == routine.status() {
			routine.setStatus(routineStatusStandby)
			go routine.run(idleQueue, p.idleTimeout)
			return true
		}
	}

	return false
}

// removeDownRoutine will remove the routine with down status
// and decrease the routinePool size
// it should only called when the pool is shuting down.
func (p *Pool) removeDownRoutine() {
	filter := p.routinePool[:0]

	for _, routine := range p.routinePool {
		if routineStatusDown != routine.status() {
			filter = append(filter, routine)
		}
	}

	p.routinePool = filter
}

// Start will push the Runnable task into the taskQueue
// which means a routine will start and work.
// A panic will occured if the pool is already shutdown, because of the taskQueue is closed.
// be careful in concurrency scenario.
func (p *Pool) Start(task Runnable) {
	p.taskQueue <- task
}

// Shutdown will stop all the routine gracefully.
// If routine is standby, it will shutdown.
// If routine is working, master routine will wait for routine finish it work by itself
// and turn into standby, then shut it down.

// Shutdown will return a [chan bool].
// A true value will push into the channel when all the routines are shuted down.
// you can wait for the popped signal to ensure all the task was handled or just ignore it.
func (p *Pool) Shutdown() chan bool {
	p.termSignal = make(chan bool)
	close(p.taskQueue)

	return p.termSignal
}
