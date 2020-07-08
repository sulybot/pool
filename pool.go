package pool

import (
	"sync"
	"time"
)

type Pool struct {
	maxRoutine  int
	expire      time.Duration
	routineList []*routine
	listStatus  [2]int
	taskCh      chan *Runnable
	reportCh    chan int
	forkLock    *sync.Mutex
}

func New(maxRoutine int, taskQueueSize int) *Pool {
	p := &Pool{
		maxRoutine:  maxRoutine,
		routineList: make([]*routine, 0, maxRoutine),
		taskCh:      make(chan *Runnable, taskQueueSize),
		reportCh:    make(chan int, maxRoutine),
		forkLock:    &sync.Mutex{},
	}

	go p.monitor()

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

func (p *Pool) Run(r *Runnable) {
	p.taskCh <- r
	if p.listStatus[P_STANDBY] <= 0 {
		p.fork()
	}
}

func (p *Pool) fork() bool {
	for _, routine := range p.routineList {
		if routine.getStatus() == R_STATUS_DOWN {
			routine.up(p.expire, p.taskCh)
			return true
		}
	}

	p.forkLock.Lock()
	defer p.forkLock.Unlock()
	if len(p.routineList) < p.maxRoutine {
		p.routineList = append(p.routineList, newRoutine(p.expire, p.reportCh, p.taskCh))
		return true
	}

	return false
}

func (p *Pool) GetStatus(status int) int {
	if 0 <= status && status < len(p.listStatus) {
		return p.listStatus[status]
	}

	return 0
}

const (
	P_STANDBY = iota
	P_WORKING
)

func (p *Pool) monitor() {
	for {
		select {
		case status, ok := <-p.reportCh:
			if !ok {
				return
			}

			switch status {
			case R_STATUS_STANDBY:
				p.listStatus[P_STANDBY]++
			case R_STATUS_RUNNING:
				p.listStatus[P_STANDBY]--
				p.listStatus[P_WORKING]++
			case R_STATUS_FINISHED:
				p.listStatus[P_WORKING]--
			case R_STATUS_DOWN:
				p.listStatus[P_STANDBY]--
			}
		}
	}
}

type Runnable interface {
	Run()
}
