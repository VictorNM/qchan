package qchan

import (
	"sync"
)

const (
	DefaultNumWorker   = 3
	DefaultMaxQueueJob = 100
	DefaultMaxTries    = 3
)

// Job define a interface for queue job
type Job interface {
	Handle() error
}

// baseJob wrap a Job with some internal field for tracking job
// Currently include field to support feature retry a job
type baseJob struct {
	Job

	tries    int
	maxTries int
}

// Queue use for dispatching a job
type Queue struct {
	maxQueueJob int
	jobChan     chan *baseJob
	maxTries    int

	numWorker  int
	workerPool chan *worker

	stopChan    chan struct{}
	stoppedChan chan struct{}
}

type Option func(q *Queue)

// WithNumWorker specify the number of workers for queue
func WithNumWorker(n int) Option {
	return func(q *Queue) {
		q.numWorker = n
	}
}

// WithMaxQueueJob specify the capacity of the queue
func WithMaxQueueJob(n int) Option {
	return func(q *Queue) {
		q.maxQueueJob = n
	}
}

// WithDefaultMaxTries specify the max number of tries before discard a job
// When an error returned from method Handle(), the job will be release back to queue,
// then the number of tries will increase by 1
// After the max number of tries is reach, the job will be discard
func WithDefaultMaxTries(n int) Option {
	return func(q *Queue) {
		q.maxTries = n
	}
}

func New(options ...Option) *Queue {
	q := &Queue{
		maxQueueJob: DefaultMaxQueueJob,
		maxTries:    DefaultMaxTries,

		numWorker: DefaultNumWorker,

		stopChan:    make(chan struct{}, 1),
		stoppedChan: make(chan struct{}, 1),
	}

	for _, option := range options {
		option(q)
	}

	q.jobChan = make(chan *baseJob, q.maxQueueJob)
	q.workerPool = make(chan *worker, q.numWorker)

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

			for i := 0; i < cap(q.workerPool); i++ {
				wg.Add(1)
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
