package pool

type Pool struct {
	idleTimeout int64
	routinePool []*routine
	taskQueue   chan Runnable
	termSignal  chan bool
}

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

	for i := 0; i < maxRoutine; i++ {
		p.routinePool[i] = new(routine)
	}

	go p.masterRoutine()
	return p
}

func (p *Pool) SetIdleTimeout(idleTimeout int64) {
	p.idleTimeout = idleTimeout
}

func (p *Pool) masterRoutine() {
	idleQueue := make(chan chan Runnable)
	tryFork := make(chan bool, 1)
	defer close(idleQueue)
	defer close(tryFork)

	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				for p.removeDownRoutine(); 0 < len(p.routinePool); p.removeDownRoutine() {
					select {
					case taskCh := <-idleQueue:
						close(taskCh)
					default:
					}
				}

				p.termSignal <- true
				return
			}
			tryFork <- true

		popRoutineLoop:
			for {
				select {
				case taskCh := <-idleQueue:
					taskCh <- task
					<-tryFork
					break popRoutineLoop

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
							break popRoutineLoop
						}
					}
				}
			}
		}
	}
}

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

func (p *Pool) allDown() bool {
	return len(p.routinePool) == 0
}

func (p *Pool) removeDownRoutine() {
	filter := p.routinePool[:0]

	for _, routine := range p.routinePool {
		if routineStatusDown != routine.status() {
			filter = append(filter, routine)
		}
	}

	p.routinePool = filter
}

func (p *Pool) Start(task Runnable) {
	p.taskQueue <- task
}

func (p *Pool) Shutdown() chan bool {
	p.termSignal = make(chan bool)
	close(p.taskQueue)

	return p.termSignal
}
