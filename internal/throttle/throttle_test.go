package throttle

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestInsertWithTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(time.Minute, 5)
		if !c.Set("default") || !c.SetWithTTL("short", time.Second, 1) ||
			!c.SetWithTTL("long", 2*time.Minute, 1) || !c.SetWithTTL("zero", 0, 1) ||
			!c.SetWithTTL("negative", -time.Second, 1) {
			t.Fatal("initial insert rejected")
		}

		if c.SetWithTTL("default", time.Second, 1) || c.Set("short") ||
			c.SetWithTTL("full", time.Second, 1) {
			t.Fatal("accepted a live key or exceeded capacity")
		}

		time.Sleep(2 * time.Second)
		if !c.SetWithTTL("short", time.Second, 1) || c.Set("default") {
			t.Fatal("override did not expire independently of the default TTL")
		}

		time.Sleep(time.Minute)
		if !c.Set("default") || !c.Set("zero") || !c.Set("negative") || c.Set("long") {
			t.Fatal("default or extended TTL was not preserved")
		}
	})
}

func TestSetWithTTLLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var (
			c        = New(time.Minute, 1)
			accepted atomic.Int32
			wg       sync.WaitGroup
		)

		for range 100 {
			wg.Go(func() {
				if c.SetWithTTL("login:admin", time.Minute, 10) {
					accepted.Add(1)
				}
			})
		}
		wg.Wait()

		if got := accepted.Load(); got != 10 {
			t.Fatalf("accepted %d attempts, want 10", got)
		}

		if c.SetWithTTL("other", time.Minute, 10) {
			t.Fatal("exceeded store capacity")
		}

		time.Sleep(59 * time.Second)
		if c.SetWithTTL("login:admin", time.Minute, 10) {
			t.Fatal("allowed attempt before window expired")
		}

		time.Sleep(time.Second)
		if !c.SetWithTTL("login:admin", time.Minute, 10) {
			t.Fatal("rejected attempt extended the window")
		}
	})
}

func TestInsert(t *testing.T) {
	f := func(capacity int) {
		t.Helper()

		c := New(time.Minute, capacity)
		for i := range capacity {
			if !c.Set(strconv.Itoa(i)) {
				t.Fatal("rejected a key before reaching capacity")
			}
		}

		// Neither existing nor new keys should succeed when it's full even on retry.
		for range 100 {
			if c.Set("0") || c.Set("new") {
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
		c.Set("expired")
		c.Set("live")
		liveExpiry := c.expires["live"]
		c.expires["expired"] = entry{expiry: time.Now().Add(-time.Minute), count: 1}
		if cleanupDue {
			c.cleanAt = time.Time{}
		}

		if got := c.Set(key); got != want {
			t.Fatalf("Insert: got %v, want %v", got, want)
		}
		if c.Set("live") || c.expires["live"] != liveExpiry {
			t.Fatal("live reservation changed")
		}
		if want && c.Set(key) {
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
				if c.Set(key) {
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
