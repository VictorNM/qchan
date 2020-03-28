package queue

type Job interface {
	Handle() error
}

type Queue struct {
	jobChan chan Job
}

func New() *Queue {
	q := &Queue{
		jobChan: make(chan Job, 1),
	}

	return q
}

func (q *Queue) Dispatch(job Job) {
	q.jobChan <- job
}

func (q *Queue) Run() {
	for {
		select {
		case job := <-q.jobChan:
			_ = job.Handle()
		}
	}
}
