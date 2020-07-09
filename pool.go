package pool

type Pool struct {
	ttl         uint
	maxRoutine  int
	taskQueue   chan Runnable
	standByCh   chan chan Runnable
	routinePool []*routine
}

func New(args ...int) *Pool {
	p := &Pool{
		ttl:        0,
		maxRoutine: 10,
		taskQueue:  make(chan Runnable),
		standByCh:  make(chan chan Runnable),
	}

	for k, v := range args {
		switch k {
		case 0: //maxRoutine
			p.maxRoutine = v
		case 1: //queueSize
			p.taskQueue = make(chan Runnable, v)
		case 2: //ttl
			p.ttl = uint(v)
		}
	}

	p.routinePool = make([]*routine, p.maxRoutine)
	for i := 0; i < p.maxRoutine; i++ {
		p.routinePool[i] = &routine{}
	}

	go p.masterRoutine()

	return p
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
			go routine.run(idleQueue, p.ttl)
			return true
		}
	}

	return false
}

func (p *Pool) Start(task Runnable) {
	p.taskQueue <- task
}
