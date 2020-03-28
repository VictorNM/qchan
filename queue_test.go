package queue_test

import (
	"queue"
	"sync"
	"testing"
	"time"
)

type mockJob struct {
	handled chan struct{}
	sleep   time.Duration
}

func (j *mockJob) Handle() error {
	time.Sleep(j.sleep)

	j.handled <- struct{}{}
	return nil
}

func (j *mockJob) assertHandleIn(t testing.TB, timeout time.Duration) {
	select {
	case <-j.handled:
		break
	case <-time.After(timeout):
		t.Errorf("not handled in %v", timeout)
	}
}

func newFastJob() *mockJob {
	return &mockJob{
		handled: make(chan struct{}),
		sleep:   0,
	}
}

func newSlowJob(sleep time.Duration) *mockJob {
	return &mockJob{
		handled: make(chan struct{}),
		sleep:   sleep,
	}
}

func newQueueAndRun(maxWorker int) *queue.Queue {
	q := queue.New(maxWorker)
	go q.Run()

	return q
}

func TestQueue_Dispatch(t *testing.T) {
	t.Run("1 fast job", func(t *testing.T) {
		t.Parallel()
		q := newQueueAndRun(1)
		j := newFastJob()

		q.Dispatch(j)

		j.assertHandleIn(t, time.Second)
	})

	t.Run("10 fast job", func(t *testing.T) {
		t.Parallel()
		q := newQueueAndRun(1)
		j := newFastJob()

		for i := 0; i < 10; i++ {
			q.Dispatch(j)
			j.assertHandleIn(t, time.Second)
		}
	})

	t.Run("1 slow job", func(t *testing.T) {
		t.Parallel()
		q := newQueueAndRun(1)
		j := newSlowJob(1 * time.Second)

		q.Dispatch(j)

		j.assertHandleIn(t, 2*time.Second)
	})

	t.Run("10 fast job with 5 worker", func(t *testing.T) {
		t.Parallel()
		q := newQueueAndRun(5)
		j := newSlowJob(10 * time.Millisecond)

		for i := 0; i < 10; i++ {
			q.Dispatch(j)
			j.assertHandleIn(t, time.Second)
		}
	})
}

type waitJob struct {
	wg *sync.WaitGroup
}

func (j *waitJob) Handle() error {
	time.Sleep(10 * time.Millisecond)

	j.wg.Done()

	return nil
}

func Benchmark_1_Worker(b *testing.B) {
	for n := 0; n < b.N; n++ {
		q := newQueueAndRun(1)

		wg := new(sync.WaitGroup)
		wg.Add(100)

		for i := 0; i < 100; i++ {
			j := &waitJob{wg : wg}
			q.Dispatch(j)
		}

		wg.Wait()
	}
}

func Benchmark_5_Worker(b *testing.B) {
	for n := 0; n < b.N; n++ {
		q := newQueueAndRun(5)
		wg := new(sync.WaitGroup)
		wg.Add(100)

		for i := 0; i < 100; i++ {
			j := &waitJob{wg : wg}
			q.Dispatch(j)
		}

		wg.Wait()
	}
}

func Benchmark_10_Worker(b *testing.B) {
	for n := 0; n < b.N; n++ {
		q := newQueueAndRun(10)
		wg := new(sync.WaitGroup)
		wg.Add(100)

		for i := 0; i < 100; i++ {
			j := &waitJob{wg : wg}
			q.Dispatch(j)
		}

		wg.Wait()
	}
}