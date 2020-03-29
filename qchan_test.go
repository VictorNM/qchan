package qchan_test

import (
	"errors"
	"fmt"
	"github.com/victornm/qchan"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TODO: custom max tries per job?
// TODO: use context for cancellation + timeout?
// TODO: if timeout: release job back to queue
// TODO: Persistent job???

func newQueueAndRun(numWorker int) *qchan.Queue {
	q := qchan.New(
		qchan.WithNumWorker(numWorker),
	)

	go q.Start()

	return q
}

type mockJob struct {
	wg *sync.WaitGroup

	handled bool
	sleep   time.Duration
}

func (j *mockJob) Handle() error {
	time.Sleep(j.sleep)
	j.handled = true

	j.wg.Done()

	return nil
}

func newMockJob(wg *sync.WaitGroup, sleep time.Duration) *mockJob {
	return &mockJob{wg: wg, handled: false, sleep: sleep}
}

func (j *mockJob) assertHandled(t testing.TB) {
	if !j.handled {
		t.Error("job was not handled")
	}
}

func TestHandleJob(t *testing.T) {
	tests := map[string]struct {
		numJobs int
		sleep   time.Duration
	}{
		"1 fast job":  {1, 0},
		"1 slow job":  {1, 10 * time.Millisecond},
		"10 fast job": {10, 0},
		"10 slow job": {10, 10 * time.Millisecond},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Given
			q := newQueueAndRun(1)
			defer q.Stop()

			// When
			jobs := run(q, test.numJobs, test.sleep)

			// Then
			for _, j := range jobs {
				j.assertHandled(t)
			}
		})
	}
}

func run(q *qchan.Queue, numJobs int, sleep time.Duration) []*mockJob {
	wg := new(sync.WaitGroup)

	var jobs []*mockJob
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		jobs = append(jobs, newMockJob(wg, sleep))
		q.Dispatch(jobs[i])
	}

	wg.Wait()

	return jobs
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

		jobDuration := 10 * time.Millisecond
		numJobs := int(time.Second / jobDuration)
		jobs := run(q, numJobs, jobDuration)

		for _, j := range jobs {
			j.assertHandled(b)
		}

		q.Stop()
	}
}

func TestStopQueue(t *testing.T) {
	origin := runtime.NumGoroutine()

	q := qchan.New(qchan.WithNumWorker(4))

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

type errorJob struct {
	*mockJob
	numHandled int
}

func newErrorJob(wg *sync.WaitGroup) *errorJob {
	return &errorJob{
		&mockJob{wg: wg},
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
			q := qchan.New(qchan.WithNumWorker(test.numWorker), qchan.WithDefaultMaxTries(test.maxTries))
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
