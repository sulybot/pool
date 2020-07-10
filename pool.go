package pool

type Pool struct {
	idleTimeout int64
	routinePool []*routine
	taskQueue   chan Runnable
}

func New( /*maxRoutine, taskQueueSize int*/ maxAndSize ...int) *Pool {
	var maxRoutine, taskQueueSize int

	for k, v := range maxAndSize {
		switch k {
		case 0: //maxRoutine
			maxRoutine = v
		case 1: //queueSize
			taskQueueSize = v
		default:
			break
		}
	}

	p := &Pool{}
	p.SetIdleTimeout(10)
	p.routinePool = make([]*routine, maxRoutine, maxRoutine)
	for i := 0; i < maxRoutine; i++ {
		p.routinePool[i] = &routine{}
	}
	p.taskQueue = make(chan Runnable, taskQueueSize)

	go p.masterRoutine()
	return p
}

func (p *Pool) SetIdleTimeout(idleTimeout int64) {
	p.idleTimeout = idleTimeout
}

func (p *Pool) masterRoutine() {
	idleQueue := make(chan chan Runnable)
	tryFork := make(chan bool, 1)

	for {
		select {
		case task := <-p.taskQueue:
			tryFork <- true

		fetchRoutineLoop:
			for {
				select {
				case taskCh := <-idleQueue:
					taskCh <- task
					<-tryFork
					break fetchRoutineLoop

				default:
					for {
						select {
						case <-tryFork:
							p.fork(idleQueue)
						case taskCh := <-idleQueue:
							taskCh <- task

							if 0 < len(tryFork) {
								<-tryFork
							}
							break fetchRoutineLoop
						}
					}
				}
			}
		}
	}
}

func (p *Pool) fork(idleQueue chan<- chan Runnable) bool {
	for _, routine := range p.routinePool {
		if ROUTINE_STATUS_DOWN == routine.status {
			routine.setStatus(ROUTINE_STATUS_STANDBY)
			go routine.run(idleQueue, p.idleTimeout)
			return true
		}
	}

	return false
}

func (p *Pool) Start(task Runnable) {
	p.taskQueue <- task
}
