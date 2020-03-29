# QCHAN

A simple queue - worker using Go channel

## Getting Started

### Features

- Queue workers
- Release back to queue if an error occur when handling job
- Max attempt on a job

### Usage example

What things you need to install the software and how to install them

```go
package main

import (
	"fmt"
	"github.com/victornm/qchan"
	"time"
)

func main() {
	q := qchan.New()

	// simulate dispatching a job
	time.AfterFunc(time.Second, func() {
		q.Dispatch(&PrintHello{})
		q.Stop() // call to end the queue
	})

	q.Start() // start the queue
	// Output: hello
}
```

## Running the tests

```bash
# Run test only
go test ./...

# Run with benchmark
go test ./... -bench=.
```