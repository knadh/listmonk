package throttle

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestInsert(t *testing.T) {
	f := func(capacity int) {
		t.Helper()

		c := New(time.Minute, capacity)
		for i := range capacity {
			if !c.Insert(strconv.Itoa(i)) {
				t.Fatal("rejected a key before reaching capacity")
			}
		}

		// Neither existing nor new keys should succeed when it's full even on retry.
		for range 100 {
			if c.Insert("0") || c.Insert("new") {
				t.Fatal("accepted a key while full")
			}
		}
		if len(c.expires) != capacity {
			t.Fatalf("store size: got %d, want %d", len(c.expires), capacity)
		}
	}

	f(1)
	f(5)
}

func TestExpiry(t *testing.T) {
	f := func(key string, cleanupDue, want bool) {
		t.Helper()

		c := New(time.Minute, 2)
		c.Insert("expired")
		c.Insert("live")
		liveExpiry := c.expires["live"]
		c.expires["expired"] = time.Now().Add(-time.Minute)
		if cleanupDue {
			c.cleanAt = time.Time{}
		}

		if got := c.Insert(key); got != want {
			t.Fatalf("Insert: got %v, want %v", got, want)
		}
		if c.Insert("live") || c.expires["live"] != liveExpiry {
			t.Fatal("live reservation changed")
		}
		if want && c.Insert(key) {
			t.Fatal("new reservation was not retained")
		}
	}

	// An expired key can be renewed before cleanup.
	f("expired", false, true)
	f("expired", true, true)

	// New keys wait until cleanup runs.
	f("new", false, false)
	f("new", true, true)
}

func TestConcurrentInsert(t *testing.T) {
	f := func(distinct bool, capacity int, want int32) {
		t.Helper()

		c := New(time.Minute, capacity)
		var accepted atomic.Int32
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := range 64 {
			wg.Go(func() {
				<-start
				key := "same"
				if distinct {
					key = strconv.Itoa(i)
				}
				if c.Insert(key) {
					accepted.Add(1)
				}
			})
		}

		close(start)
		wg.Wait()
		if got := accepted.Load(); got != want {
			t.Fatalf("accepted: got %d, want %d", got, want)
		}
		if len(c.expires) != int(want) {
			t.Fatalf("stored: got %d, want %d", len(c.expires), want)
		}
	}

	f(false, 5, 1)
	f(true, 5, 5)
}
