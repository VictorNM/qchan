package qchan

import (
	"log"
	"sync"
)

const (
	DefaultNumWorkers = 10
	DefaultMaxQueuedJob = 100
)

type job struct {
	name string
	data []byte
}

type HandleFunc func(data []byte)

type Queue struct {
	jobChan    chan job
	wJobChan   chan job
	handlerMap map[string]HandleFunc

	numWorkers int
	workerPool chan *worker

	stopChan chan struct{}
	wg       *sync.WaitGroup
}

func New() *Queue {
	q := &Queue{
		jobChan:    make(chan job, DefaultMaxQueuedJob),
		wJobChan:   make(chan job, DefaultNumWorkers),
		handlerMap: make(map[string]HandleFunc),

		numWorkers: DefaultNumWorkers,

		stopChan: make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}

	q.workerPool = make(chan *worker, q.numWorkers)
	q.wg.Add(1)

	return q
}

func (q *Queue) Enqueue(name string, data []byte) {
	q.jobChan <- job{
		name: name,
		data: data,
	}
}

func (q *Queue) SetHandler(name string, handlerFunc HandleFunc) {
	q.handlerMap[name] = handlerFunc
}

func (q *Queue) Start() {
	for {
		select {
		case j := <-q.jobChan:
			if len(q.workerPool) < q.numWorkers {
				w := newWorker(q)
				q.workerPool <- w
				go w.start()
			}
			q.wJobChan <- j
		case <-q.stopChan:
			log.Println("stop queue")
			q.wg.Done()
			return
		}
	}
}

func (q *Queue) Stop() {
	close(q.stopChan)
	q.wg.Wait()
}

// WORKER
type worker struct {
	id      int
	q       *Queue
	jobChan chan job
}

func newWorker(q *Queue) *worker {
	return &worker{
		q:       q,
		jobChan: make(chan job, 1),
	}
}

func (w *worker) start() {
	log.Println("start worker")
	w.q.wg.Add(1)
	for {
		select {
		case job := <-w.q.wJobChan:
			f := w.q.handlerMap[job.name]
			if f != nil {
				f(job.data)
			}
		case <-w.q.stopChan:
			log.Println("stop worker")
			w.q.wg.Done()
			return
		}
	}
}
