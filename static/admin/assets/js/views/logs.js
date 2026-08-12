import { urls } from '../main.js';

// Regexp for splitting a log line into [timestamp] [file] [message].
// 2021/05/01 00:00:00:00 init.go:99: reading config: config.toml
const reLine = /^([0-9\s:/]+\.[0-9]{6}) (.+?\.go:[0-9]+|\*):\s(.+)$/;

const POLL_INTERVAL = 10000;

function splitLine(l) {
  const parts = l.split(reLine);
  if (parts.length !== 5) {
    return { timestamp: '', file: '', message: l };
  }
  return { timestamp: parts[1], file: parts[2], message: parts[3] };
}

function component() {
  return {
    lines: (window._logs || []).filter((l) => l).map(splitLine),
    loading: false,
    pollID: null,

    init() {
      this.$nextTick(() => this.scrollToBottom());

      // Refresh the logs every 10 seconds.
      this.pollID = setInterval(() => this.getLogs(), POLL_INTERVAL);
    },

    destroy() {
      clearInterval(this.pollID);
    },

    async getLogs() {
      this.loading = true;
      try {
        const resp = await fetch(`${urls.api}/logs`, { headers: { Accept: 'application/json' } });
        const out = await resp.json();
        this.lines = (out.data || []).filter((l) => l).map(splitLine);
        this.$nextTick(() => this.scrollToBottom());
      } finally {
        this.loading = false;
      }
    },

    scrollToBottom() {
      if (this.$refs.lines) {
        this.$refs.lines.scrollTop = this.$refs.lines.scrollHeight;
      }
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('logsView', component);
}, { once: true });
