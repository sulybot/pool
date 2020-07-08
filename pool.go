package pool

import "time"

type Pool struct {
	maxRoutine  int
	expire      time.Duration
	routineList []*routine
	taskCh      chan *Runnable
}

func New(maxRoutine int, taskQueueSize int) *Pool {
	p := &Pool{
		maxRoutine:  maxRoutine,
		routineList: make([]*routine, 0, maxRoutine),
		taskCh:      make(chan *Runnable, taskQueueSize),
	}

	return p
}

func (p *Pool) SetExpire(expire string) error {
	ex, err := time.ParseDuration(expire)
	if nil != err {
		return err
	}

	p.expire = ex
	return nil
}

func (p *Pool) Run(r *Runnable) error {
	p.taskCh <- r
}

func (p *Pool) fork() {

}

type Runnable interface {
	Run()
}
