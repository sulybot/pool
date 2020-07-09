package pool

type Pool struct {
	ttl         uint
	maxRoutine  int
	taskCh      chan Runnable
	standByCh   chan chan Runnable
	routinePool []*routine
}

func New(args ...int) *Pool {
	p := &Pool{
		ttl:        0,
		maxRoutine: 10,
		taskCh:     make(chan Runnable),
		standByCh:  make(chan chan Runnable),
	}

	for k, v := range args {
		switch k {
		case 0: //maxRoutine
			p.maxRoutine = v
		case 1: //queueSize
			p.taskCh = make(chan Runnable, v)
		case 2: //ttl
			p.ttl = uint(v)
		}
	}

	p.routinePool = make([]*routine, p.maxRoutine)
	for i := 0; i < p.maxRoutine; i++ {
		p.routinePool[i] = &routine{}
	}

	go p.master()

	return p
}

func (p *Pool) master() {
	for {
		newRoutineForked := false
		select {
		case task := <-p.taskCh:
		forkLoop:
			for {
				select {
				case taskCh := <-p.standByCh:
					taskCh <- task
					break forkLoop
				default:
					if !newRoutineForked {
						for _, routine := range p.routinePool {
							if ROUTINE_STATUS_DOWN == routine.status {
								go routine.run(p.standByCh, p.ttl)
								newRoutineForked = true
							}
						}
					}
				}
			}
		}
	}
}

func (p *Pool) Start(task Runnable) {
	p.taskCh <- task
}
