package tun

import (
	"runtime"
	"testing"

	"github.com/xtls/xray-core/common/buf"
)

const benchBatch = 64

// Raw cost of queueing and draining a small burst, as one flow's reader does.
func BenchmarkQueueBurstChan(b *testing.B) {
	ch := make(chan *buf.Buffer, udpQueueLimit)
	p := &buf.Buffer{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for range benchBatch {
			ch <- p
		}
		for range benchBatch {
			<-ch
		}
	}
}

func BenchmarkQueueBurstPacketQueue(b *testing.B) {
	q := newPacketQueue(udpQueueLimit)
	p := &buf.Buffer{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for range benchBatch {
			q.push(p)
		}
		for range benchBatch {
			q.pop()
		}
	}
}

// Producer and consumer on different goroutines; the producer yields when the
// queue is full instead of spinning, like a blocking channel send would.
func BenchmarkQueueStreamChan(b *testing.B) {
	ch := make(chan *buf.Buffer, udpQueueLimit)
	p := &buf.Buffer{}
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- p
	}
	close(ch)
	<-done
}

func BenchmarkQueueStreamPacketQueue(b *testing.B) {
	q := newPacketQueue(udpQueueLimit)
	p := &buf.Buffer{}
	done := make(chan struct{})
	go func() {
		for {
			if _, ok := q.pop(); !ok {
				break
			}
		}
		close(done)
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for !q.push(p) {
			runtime.Gosched()
		}
	}
	q.close()
	<-done
}
