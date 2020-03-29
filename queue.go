package queue

import (
	"sync"
)

// Job define a interface for queue job
type Job interface {
	Handle() error
}

type baseJob struct {
	Job

	tries    int
	maxTries int
}

// Queue use for dispatching a job
type Queue struct {
	jobChan  chan *baseJob
	maxTries int

	numWorker  int
	workerPool chan *worker

	stopChan    chan struct{}
	stoppedChan chan struct{}
}

func New(numWorker int, maxQueueJob int, defaultMaxTries int) *Queue {
	q := &Queue{
		jobChan:  make(chan *baseJob, maxQueueJob),
		maxTries: defaultMaxTries,

		numWorker:  numWorker,
		workerPool: make(chan *worker, numWorker),

		stopChan:    make(chan struct{}, 1),
		stoppedChan: make(chan struct{}, 1),
	}

	return q
}

// Dispatch is for dispatching a job to queue
func (q *Queue) Dispatch(job Job) {
	q.jobChan <- &baseJob{
		Job:      job,
		tries:    0,
		maxTries: q.maxTries,
	}
}

func (q *Queue) Start() {
	for i := 0; i < q.numWorker; i++ {
		worker := newWorker(q)
		q.workerPool <- worker
		go worker.start()
	}

	for {
		select {
		case job := <-q.jobChan:
			if job.tries >= job.maxTries {
				break
			}
			w := <-q.workerPool
			job.tries++
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

func (q *Queue) releaseWorker(w *worker) {
	q.workerPool <- w
}

func (q *Queue) releaseJob(job Job) {
	q.jobChan <- job.(*baseJob)
}

// WORKER
type worker struct {
	q       *Queue
	jobChan chan Job

	stopChan    chan struct{}
	stoppedChan chan struct{}
}

func newWorker(q *Queue) *worker {
	return &worker{
		q:           q,
		jobChan:     make(chan Job, 1),
		stopChan:    make(chan struct{}, 1),
		stoppedChan: make(chan struct{}, 1),
	}
}

func (w *worker) start() {
	for {
		select {
		case job := <-w.jobChan:
			if err := job.Handle(); err != nil {
				w.q.releaseJob(job)
			}
		case <-w.stopChan:
			w.stoppedChan <- struct{}{}
			return
		}
		w.q.releaseWorker(w)
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
