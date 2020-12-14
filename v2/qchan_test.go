package qchan_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/victornm/qchan/v2"
)

type testJob struct {
	ID int
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
		"100 slow job": {100, 10 * time.Millisecond},
		"1000 slow job": {1000, 10 * time.Millisecond},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			q := qchan.New()
			defer q.Stop()

			go func() {
				q.Start()
			}()

			c := make(chan int, test.numJobs)
			q.SetHandler("test",func(data []byte) {
				var j testJob
				err := json.Unmarshal(data, &j)
				if err != nil {
					t.Fatalf("unmarshal data: %v", err)
				}
				t.Logf("received id: %v", j.ID)
				time.Sleep(test.sleep)
				c <- j.ID
			})

			for i := 0; i < test.numJobs; i++ {
				data, _ := json.Marshal(testJob{ID: i})
				q.Enqueue("test", data)
			}

			var ids []int
			for id := range c {
				ids = append(ids, id)
				if len(ids) == test.numJobs {
					close(c)
				}
			}

			t.Logf("duration: %v", time.Now().Sub(start))
		})
	}
}