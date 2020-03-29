package queue

import (
	"sync"
)

type Job interface {
	Handle() error
}

type Queue struct {
	jobChan chan Job

	numWorker  int
	workerPool chan *worker

	stopChan    chan struct{}
	stoppedChan chan struct{}
}

func New(numWorker int, maxQueueJob int) *Queue {
	q := &Queue{
		jobChan:    make(chan Job, maxQueueJob),
		numWorker:  numWorker,
		workerPool: make(chan *worker, numWorker),

		stopChan:    make(chan struct{}, 1),
		stoppedChan: make(chan struct{}, 1),
	}

	return q
}

func (q *Queue) Dispatch(job Job) {
	q.jobChan <- job
}

func (q *Queue) Start() {
	for i := 0; i < q.numWorker; i++ {
		worker := newWorker(q.workerPool)
		q.workerPool <- worker
		go worker.start()
	}

	for {
		select {
		case job := <-q.jobChan:
			w := <-q.workerPool
			w.consume(job)
		case <-q.stopChan:
			wg := new(sync.WaitGroup)
			wg.Add(q.numWorker)

			for i := 0; i < q.numWorker; i++ {
				w := <-q.workerPool
				w.stop(wg)
			}
			wg.Wait()

			q.stoppedChan <- struct{}{}
			return
		}
	}
}

func (q *Queue) Stop() {
	q.stopChan <- struct{}{}
	<-q.stoppedChan
}

type worker struct {
	poolChan chan *worker
	jobChan  chan Job

	stopChan    chan struct{}
	stoppedChan chan struct{}
}

func newWorker(poolChan chan *worker) *worker {
	return &worker{
		poolChan:    poolChan,
		jobChan:     make(chan Job, 1),
		stopChan:    make(chan struct{}, 1),
		stoppedChan: make(chan struct{}, 1),
	}
}

func (w *worker) start() {
	for {
		select {
		case job := <-w.jobChan:
			_ = job.Handle()
		case <-w.stopChan:
			w.stoppedChan <- struct{}{}
			return
		}
		w.poolChan <- w
	}
}

func (w *worker) consume(job Job) {
	w.jobChan <- job
}

func (w *worker) stop(wg *sync.WaitGroup) {
	w.stopChan <- struct{}{}
	<-w.stoppedChan
	wg.Done()
}
