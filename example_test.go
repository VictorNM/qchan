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

	go q.Start() // start the queue


	q.Dispatch(&FailJob{wg: wg})
	wg.Wait()

	q.Stop()

	// Output: hello
	// hello
	// hello
	// hello
	// hello
}
