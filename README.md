# QCHAN

A simple queue - worker using Go channel

## Getting Started

### Features

- Worker pool, limit the number of goroutines
- Release back to queue if an error occur when handling job
- Max attempt on a job

### Usage example

- Simple use case

```go
package qchan_test

import (
	"errors"
	"fmt"
	"github.com/victornm/qchan"
	"sync"
)

type PrintHello struct {
	wg *sync.WaitGroup
}

func (j *PrintHello) Handle() error {
	fmt.Println("hello")
	j.wg.Done()
	return nil
}

func ExampleSimpleUseCase() {
	q := qchan.New(qchan.WithDefaultMaxTries(5))
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// start the queue
	go q.Start()

	// dispatch a job to queue
	q.Dispatch(&PrintHello{wg: wg})
	wg.Wait()

	// stop the queue
	q.Stop()
	// Output: hello
}
```

- Retry job

```go
package qchan_test

import (
	"errors"
	"fmt"
	"github.com/victornm/qchan"
	"sync"
)

type FailJob struct {
	wg *sync.WaitGroup
}

func (j *FailJob) Handle() error {
	fmt.Println("hello")
	j.wg.Done()
	return errors.New("error")
}

func ExampleRetryJob() {
	q := qchan.New(qchan.WithDefaultMaxTries(5))
	wg := &sync.WaitGroup{}
	wg.Add(5)

	go q.Start()

	q.Dispatch(&FailJob{wg: wg})
	wg.Wait()

	q.Stop()
	// Output: hello
	// hello
	// hello
	// hello
	// hello
}
```

## Running the tests

```bash
# Run test only
go test ./...

# Run with benchmark
go test ./... -bench=.
```