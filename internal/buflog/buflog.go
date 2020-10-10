package buflog

import (
	"bytes"
	"strings"
	"sync"
)

// BufLog implements a simple log buffer that can be supplied to a std
// log instance. It stores logs up to N lines.
type BufLog struct {
	maxLines int
	buf      *bytes.Buffer
	lines    []string

	sync.RWMutex
}

// New returns a new log buffer that stores up to maxLines lines.
func New(maxLines int) *BufLog {
	return &BufLog{
		maxLines: maxLines,
		buf:      &bytes.Buffer{},
		lines:    make([]string, 0, maxLines),
	}
}

// Write writes a log item to the buffer maintaining maxLines capacity
// using LIFO.
func (bu *BufLog) Write(b []byte) (n int, err error) {
	bu.Lock()
	if len(bu.lines) >= bu.maxLines {
		bu.lines[0] = ""
		bu.lines = bu.lines[1:len(bu.lines)]
	}

	bu.lines = append(bu.lines, strings.TrimSpace(string(b)))
	bu.Unlock()
	return len(b), nil
}

// Lines returns the log lines.
func (bu *BufLog) Lines() []string {
	bu.RLock()
	defer bu.RUnlock()

	out := make([]string, len(bu.lines))
	copy(out[:], bu.lines[:])
	return out
}
