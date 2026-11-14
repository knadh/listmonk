package tmptokens

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func storedToken(id string) (Token, bool) {
	mu.RLock()
	defer mu.RUnlock()

	v, ok := tokens[id]
	return v, ok
}

func TestSet(t *testing.T) {
	f := func(data any) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		Set(id, time.Hour, "old")
		Check(id)

		// Replacing a token also resets its attempt count.
		Set(id, time.Minute, data)
		v, _ := storedToken(id)
		if v.Data != data || v.TTL != time.Minute || v.Count != 0 {
			t.Fatalf("replacement: got %+v", v)
		}
	}

	f("new")
	f(42)
	f(nil)
}

func TestSetNX(t *testing.T) {
	f := func(ttl time.Duration, checks int, data, want any) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		if got := SetNX(id, ttl, data); got != data {
			t.Fatalf("first SetNX: got %v, want %v", got, data)
		}

		for range checks {
			Check(id)
		}

		before, _ := storedToken(id)
		if got := SetNX(id, time.Hour, "new"); got != want {
			t.Fatalf("second SetNX: got %v, want %v", got, want)
		}

		after, _ := storedToken(id)
		if ttl > 0 && after != before {
			t.Fatalf("live token changed: before %+v, after %+v", before, after)
		}
		if ttl < 0 && (after.Data != want || after.Count != 0 || after.TTL != time.Hour) {
			t.Fatalf("expired token not replaced: %+v", after)
		}
	}

	f(time.Minute, 0, "old", "old")
	f(time.Minute, 0, nil, nil)

	// Exhaustion must not reset the budget.
	f(time.Minute, maxTries+1, "old", "old")

	f(-time.Minute, 0, "old", "new")
}

func TestReserve(t *testing.T) {
	f := func(ttl time.Duration, checks int, want bool) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		if !Reserve(id, ttl) {
			t.Fatal("first reservation failed")
		}

		for range checks {
			Check(id)
		}

		before, _ := storedToken(id)
		if got := Reserve(id, time.Hour); got != want {
			t.Fatalf("second Reserve: got %v, want %v", got, want)
		}

		after, _ := storedToken(id)
		if !want && after != before {
			t.Fatal("rejected reservation changed the token")
		}
		if Reserve(id, time.Hour) {
			t.Fatal("reservation was not retained")
		}
	}

	f(time.Minute, 0, false)

	// Exhausting Check must not allow another reservation.
	f(time.Minute, maxTries+1, false)

	f(-time.Minute, 0, true)
}

func TestReserveExisting(t *testing.T) {
	f := func(data any) {
		t.Helper()

		id := t.Name()
		defer Delete(id)
		Set(id, time.Minute, data)
		before, _ := storedToken(id)

		if Reserve(id, time.Hour) {
			t.Fatal("reserved an existing token")
		}
		if after, _ := storedToken(id); after != before {
			t.Fatal("reservation changed the existing token")
		}
	}

	// Reservations share the namespace with ordinary tokens.
	f("secret")
	f(42)
	f(nil)
}

func TestCheck(t *testing.T) {
	f := func(data any) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		Set(id, time.Minute, data)

		for i := range maxTries {
			if got, err := Check(id); err != nil || got != data {
				t.Fatalf("check %d: got (%v, %v), want (%v, nil)", i+1, got, err, data)
			}
		}

		// Exhausted tokens are rejected but remain in the store.
		for range 2 {
			if got, err := Check(id); got != nil || err != Err {
				t.Fatalf("exhausted check: got (%v, %v), want (nil, Err)", got, err)
			}
		}
		if v, ok := storedToken(id); !ok || v.Count != maxTries {
			t.Fatalf("exhausted token not retained: %+v", v)
		}
	}

	f("secret")
	f(42)
	f(nil)
}

func TestGet(t *testing.T) {
	f := func(data any, checks int) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		Set(id, time.Minute, data)

		for range checks {
			Check(id)
		}

		if got, err := Get(id); got != data || err != nil {
			t.Fatalf("Get: got (%v, %v), want (%v, nil)", got, err, data)
		}
		if got, err := Get(id); got != nil || err != Err {
			t.Fatalf("second Get: got (%v, %v), want (nil, Err)", got, err)
		}
	}

	f("secret", 0)
	f(nil, 0)

	// Get does not increment count.
	f(42, maxTries+1)
}

func TestMissingAndExpired(t *testing.T) {
	f := func(read func(string) (any, error)) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		if got, err := read(id); got != nil || err != Err {
			t.Fatalf("missing token: got (%v, %v), want (nil, Err)", got, err)
		}

		Set(id, -time.Minute, "expired")
		if got, err := read(id); got != nil || err != Err {
			t.Fatalf("expired token: got (%v, %v), want (nil, Err)", got, err)
		}
		if _, ok := storedToken(id); ok {
			t.Fatal("expired token was not deleted")
		}
	}

	f(Check)
	f(Get)
}

func TestMatch(t *testing.T) {
	f := func(data any, value string, ttl time.Duration, consume, want bool) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		if Match(id, value, consume) {
			t.Fatal("missing token matched")
		}

		Set(id, ttl, data)
		before, _ := storedToken(id)
		if got := Match(id, value, consume); got != want {
			t.Fatalf("Match: got %v, want %v", got, want)
		}

		after, exists := storedToken(id)
		if ttl < 0 || (want && consume) {
			if exists {
				t.Fatal("token was not deleted")
			}
		} else if !exists || after != before {
			t.Fatal("Match changed an unconsumed token")
		}
	}

	f("secret", "secret", time.Minute, false, true)
	f("secret", "secret", time.Minute, true, true)

	// A wrong value must not consume the token.
	f("secret", "wrong", time.Minute, false, false)
	f("secret", "wrong", time.Minute, true, false)

	f("secret", "secret", -time.Minute, true, false)

	// Only string data can match.
	f(42, "42", time.Minute, true, false)
	f(nil, "", time.Minute, true, false)
	f("", "", time.Minute, true, true)
}

func TestMatchAfterChecks(t *testing.T) {
	f := func(consume bool) {
		t.Helper()

		id := t.Name()
		defer Delete(id)
		Set(id, time.Minute, "secret")
		for range maxTries + 1 {
			Check(id)
		}

		// Match is independent of the Check budget.
		if !Match(id, "secret", consume) {
			t.Fatal("exhausted token did not match")
		}
		if _, exists := storedToken(id); exists == consume {
			t.Fatalf("token present: got %v, want %v", exists, !consume)
		}
	}

	f(false)
	f(true)
}

func TestDeleteAndClean(t *testing.T) {
	f := func(remove func(string), ttl time.Duration, wantPresent bool) {
		t.Helper()

		id := t.Name()
		other := id + ":live"
		defer Delete(id)
		defer Delete(other)

		Set(other, time.Hour, "keep")
		before, _ := storedToken(other)

		remove(id) // Missing IDs shouldn't be an error.
		Set(id, ttl, "value")
		remove(id)
		if _, ok := storedToken(id); ok != wantPresent {
			t.Fatalf("token present: got %v, want %v", ok, wantPresent)
		}
		if after, ok := storedToken(other); !ok || after != before {
			t.Fatal("unrelated live token changed")
		}
	}

	f(Delete, time.Minute, false)
	f(Delete, -time.Minute, false)

	// Cleanup should retain live tokens.
	clean := func(string) { Clean() }
	f(clean, time.Minute, true)
	f(clean, -time.Minute, false)
}

func TestConcurrent(t *testing.T) {
	f := func(run func(string) bool, want int32) {
		t.Helper()

		id := t.Name()
		defer Delete(id)

		var successes atomic.Int32
		var wg sync.WaitGroup
		start := make(chan struct{})
		for range 64 {
			wg.Go(func() {
				<-start
				if run(id) {
					successes.Add(1)
				}
			})
		}

		close(start)
		wg.Wait()
		if got := successes.Load(); got != want {
			t.Fatalf("successful calls: got %d, want %d", got, want)
		}
	}

	// Only one caller can create the token.
	f(func(id string) bool { return Reserve(id, time.Minute) }, 1)
	f(func(id string) bool {
		value := new(int)
		return SetNX(id, time.Minute, value) == value
	}, 1)

	// Concurrent checks share the same attempts.
	f(func(id string) bool {
		SetNX(id, time.Minute, "secret")
		_, err := Check(id)
		return err == nil
	}, maxTries)

	// Only one caller can consume the token.
	Set(t.Name(), time.Minute, "secret")
	f(func(id string) bool { return Match(id, "secret", true) }, 1)

	Set(t.Name(), time.Minute, "secret")
	f(func(id string) bool { _, err := Get(id); return err == nil }, 1)
}
