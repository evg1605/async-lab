package workers_pool

import (
	"errors"
	"sync"
)

var (
	ErrAddAfterClose = errors.New("error adding job after close")
)

type WorkersPool struct {
	poolSize int
	wg       *sync.WaitGroup
	jobs     []func()
	newJobs  chan func()
	closed   chan struct{}
}

func StartNewPool(poolSize int) *WorkersPool {
	wp := &WorkersPool{
		poolSize: poolSize,
		wg:       &sync.WaitGroup{},
		newJobs:  make(chan func()),
		closed:   make(chan struct{}),
	}
	wp.wg.Add(wp.poolSize)
	go wp.start()
	return wp
}

func (wp *WorkersPool) AddJob(job func()) error {
	select {
	case wp.newJobs <- job:
		return nil
	case <-wp.closed:
		return ErrAddAfterClose
	}
}

func (wp *WorkersPool) WaitJobsAndStop() {
	close(wp.closed)
	wp.wg.Wait()
}

func (wp *WorkersPool) start() {
	workerJobs := make(chan func())

	for i := 0; i < wp.poolSize; i++ {
		go wp.worker(workerJobs)
	}

	closed := wp.closed
	newJobs := wp.newJobs
	for {
		var nextJob func()
		var lWorkerJobs chan func()

		if len(wp.jobs) > 0 {
			nextJob = wp.jobs[0]
			lWorkerJobs = workerJobs
		} else if closed == nil {
			close(workerJobs)
			return
		}

		select {
		case lWorkerJobs <- nextJob:
			wp.jobs = wp.jobs[1:]
		case j := <-newJobs:
			wp.jobs = append(wp.jobs, j)
		case <-closed:
			closed = nil
			newJobs = nil
		}
	}
}

func (wp *WorkersPool) worker(workerJobs chan func()) {
	defer wp.wg.Done()

	for j := range workerJobs {
		j()
	}
}
