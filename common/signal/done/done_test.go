package done

import (
	"sync"
	"testing"
)

func TestConcurrentClose(t *testing.T) {
	d := New()
	if d.Done() {
		t.Fatal("new instance is already done")
	}
	wait := d.Wait()
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			if err := d.Close(); err != nil {
				t.Error(err)
			}
			if !d.Done() {
				t.Error("Close returned before completion")
			}
		})
	}
	wg.Wait()
	select {
	case <-wait:
	default:
		t.Fatal("wait channel is not closed")
	}
}
