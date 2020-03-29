package queue_test

import (
	"errors"
	"fmt"
	"queue"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TODO: custom max tries per job?
// TODO: use context for cancellation + timeout?
// TODO: if timeout: release job back to queue

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

func newQueueAndRun(numWorker int) *queue.Queue {
	q := queue.New(
		queue.WithNumWorker(numWorker),
	)

	go q.Start()

	return q
}

func TestQueue_Dispatch(t *testing.T) {
	t.Run("1 fast job", func(t *testing.T) {
		q := newQueueAndRun(1)
		j := newFastJob()

		q.Dispatch(j)

		j.assertHandleIn(t, time.Second)

		q.Stop()
	})

	t.Run("1 slow job", func(t *testing.T) {
		q := newQueueAndRun(1)
		j := newSlowJob(1 * time.Second)

		q.Dispatch(j)

		j.assertHandleIn(t, 2*time.Second)

		q.Stop()
	})
}

type errorJob struct {
	*waitJob
	numHandled int
}

func newErrorJob(wg *sync.WaitGroup) *errorJob {
	return &errorJob{
		&waitJob{wg: wg},
		0,
	}
}

func (j *errorJob) assertNumHandled(t testing.TB, n int) {
	if n != j.numHandled {
		t.Errorf("wanted %d but got %d", n, j.numHandled)
	}
}

func (j *errorJob) Handle() error {
	time.Sleep(10 * time.Millisecond)
	j.numHandled++
	j.wg.Done()
	return errors.New("error")
}

func TestReleaseAndMaxTries(t *testing.T) {
	tests := []struct {
		maxTries  int
		numWorker int
	}{
		{0, 1},
		{1, 1},
		{5, 1},
		{10, 1},
		{5, 2},
	}

	for _, test := range tests {
		name := fmt.Sprintf(
			"Max tries = %d; Num worker = %d",
			test.maxTries,
			test.numWorker,
		)
		t.Run(name, func(t *testing.T) {
			q := queue.New(queue.WithNumWorker(test.numWorker), queue.WithMaxTries(test.maxTries))
			go q.Start()
			defer q.Stop()

			wg := new(sync.WaitGroup)
			wg.Add(test.maxTries)

			j := newErrorJob(wg)

			q.Dispatch(j)

			wg.Wait()
		})
	}
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
	benchmarkWithNWorker(b, 1)
}

func Benchmark_5_Worker(b *testing.B) {
	benchmarkWithNWorker(b, 5)
}

func Benchmark_10_Worker(b *testing.B) {
	benchmarkWithNWorker(b, 10)
}

func benchmarkWithNWorker(b *testing.B, numWorker int) {
	for n := 0; n < b.N; n++ {
		q := newQueueAndRun(numWorker)
		wg := new(sync.WaitGroup)
		wg.Add(100)

		for i := 0; i < 100; i++ {
			j := &waitJob{wg: wg}
			q.Dispatch(j)
		}

		wg.Wait()

		q.Stop()
	}
}

func TestStopQueue(t *testing.T) {
	origin := runtime.NumGoroutine()

	q := queue.New(queue.WithNumWorker(4))

	wait := make(chan struct{})
	time.AfterFunc(time.Second, func() {
		q.Stop()
		wait <- struct{}{}
	})
	q.Start()

	<-wait

	current := runtime.NumGoroutine()
	if origin != current {
		t.Errorf("wanted %d but got %d\n", origin, current)
	}
}