package queue

type Job interface {
	Handle() error
}

type Queue struct {
	jobChan chan Job

	maxWorker  int
	workerPool chan *worker
}

func New(maxWorker int) *Queue {
	q := &Queue{
		jobChan:    make(chan Job, 100),
		maxWorker:  maxWorker,
		workerPool: make(chan *worker, maxWorker),
	}

	return q
}

func (q *Queue) Dispatch(job Job) {
	q.jobChan <- job
}

func (q *Queue) Run() {
	for i := 0; i < q.maxWorker; i++ {
		worker := newWorker(q.workerPool)
		q.workerPool <- worker
		go worker.start()
	}

	for {
		w := <-q.workerPool

		select {
		case job := <-q.jobChan:
			w.consume(job)
		}
	}
}

type worker struct {
	id       int
	poolChan chan *worker
	jobChan  chan Job
}

func newWorker(poolChan chan *worker) *worker {
	return &worker{poolChan: poolChan, jobChan: make(chan Job, 1)}
}

func (w *worker) start() {
	for {
		select {
		case job := <-w.jobChan:
			_ = job.Handle()
		}
		w.poolChan <- w
	}
}

func (w *worker) consume(job Job) {
	w.jobChan <- job
}
