package manager

import "time"

// slidingWindow limits the number of messages sent in any rolling window of
// time, eg: no more than 100 messages in any 60 minutes. It keeps the send
// times of the messages in the current window and on reaching the limit,
// holds further messages until the oldest one moves out of the window.
type slidingWindow struct {
	rate     int
	duration time.Duration

	// Send times of the messages in the window, oldest first.
	sent []time.Time
}

// wait forgets messages that have moved out of the window and returns how long
// to wait from now before the next message can be sent. rate should be > 0.
func (w *slidingWindow) wait(now time.Time) time.Duration {
	cutoff := now.Add(-w.duration)
	n := 0
	for n < len(w.sent) && !w.sent[n].After(cutoff) {
		n++
	}
	w.sent = w.sent[n:]

	if len(w.sent) < w.rate {
		return 0
	}

	// Wait for enough of the oldest messages to move out of the window.
	return w.sent[len(w.sent)-w.rate].Add(w.duration).Sub(now)
}

// add records a message sent at t.
func (w *slidingWindow) add(t time.Time) {
	w.sent = append(w.sent, t)
}
