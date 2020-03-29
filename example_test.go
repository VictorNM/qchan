package qchan_test

import (
	"fmt"
	"github.com/victornm/qchan"
	"time"
)

type PrintHello struct {
}

func (j *PrintHello) Handle() error {
	fmt.Println("hello")
	return nil
}

func ExampleSimpleUseCase() {
	q := qchan.New()

	// simulate dispatching a job
	time.AfterFunc(time.Second, func() {
		q.Dispatch(&PrintHello{})
		q.Stop() // call to end the queue
	})

	q.Start() // start the queue
	// Output: hello
}
