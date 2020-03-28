package queue_test

import (
	"queue"
	"testing"
	"time"
)

type testJob struct {
	handled chan struct{}
}

func (j *testJob) Handle() error {
	j.handled <- struct{}{}
	return nil
}

func (j *testJob) assertHandleIn(t *testing.T, timeout time.Duration) {
	select {
	case <-j.handled:
		break
	case <-time.After(timeout):
		t.Errorf("not handled in %v", timeout)
	}
}

func newJob() *testJob {
	return &testJob{handled: make(chan struct{})}
}

func newQueueAndRun() * queue.Queue {
	q := queue.New()
	go q.Run()

	return q
}

func TestQueue_Dispatch(t *testing.T) {
	q := newQueueAndRun()
	j := newJob()

	q.Dispatch(j)

	j.assertHandleIn(t, time.Second)
}
