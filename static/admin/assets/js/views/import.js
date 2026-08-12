import {
  urls,
  i18n,
  ListTag,
} from '../main.js';
import * as u from '../utils.js';

// Regexp for splitting a log line into [timestamp] [file] [message].
// 2021/05/01 00:00:00:00 init.go:99: reading config: config.toml
const reLine = /^([0-9\s:/]+\.[0-9]{6}) (.+?\.go:[0-9]+|\*):\s(.+)$/;

function component() {
  return {
    form: {
      mode: 'subscribe',
      subStatus: 'unconfirmed',
      delim: ',',
      lists: (window._selectedLists || []).map((l) => new ListTag(l)),
      overwriteUserInfo: false,
      overwriteSubStatus: false,
      file: null,
    },

    isProcessing: false,
    status: window._importStatus || { status: 'none' },
    logLines: [],
    pollID: null,

    init() {
      // Reset the status radio whenever the mode changes.
      this.$watch('form.mode', (mode) => {
        this.form.subStatus = mode === 'subscribe' ? 'unconfirmed' : 'unsubscribed';
      });

      // Start polling if the upload's running.
      if (this.isRunning) {
        this.pollStatus();
      } else if (this.isDone) {
        this.getLogs();
      }
    },

    // ===============
    // State getters.
    get isRunning() {
      return this.status.status === 'importing' || this.status.status === 'stopping';
    },

    get isDone() {
      return ['finished', 'stopped', 'failed'].includes(this.status.status);
    },

    get canUpload() {
      return !!this.form.file && !(this.form.mode === 'subscribe' && this.form.lists.length === 0);
    },

    get progress() {
      if (!this.status || !(this.status.total > 0)) {
        return 0;
      }
      return Math.ceil((this.status.imported / this.status.total) * 100);
    },

    get recordsLabel() {
      return i18n.ts('import.recordsCount', { num: this.status.imported || 0, total: this.status.total || 0 });
    },

    // Badge colour for the current status.
    get statusVariant() {
      if (this.status.status === 'finished') {
        return 'success';
      }
      if (['failed', 'stopped'].includes(this.status.status)) {
        return 'danger';
      }

      return 'secondary';
    },

    // ===============
    // Event handlers.
    onUpload() {
      if (this.form.mode === 'subscribe' && this.form.overwriteSubStatus) {
        u.confirm(i18n.t('import.subscribeWarning')).then((ok) => {
          if (ok) {
            this.onSubmit();
          }
        });
        return;
      }
      this.onSubmit();
    },

    async onSubmit() {
      this.isProcessing = true;

      const params = new FormData();
      params.set('params', JSON.stringify({
        mode: this.form.mode,
        subscription_status: this.form.subStatus,
        delim: this.form.delim,
        lists: this.form.lists.map((l) => l.id),
        overwrite_userinfo: this.form.overwriteUserInfo,
        overwrite_subscription_status: this.form.overwriteSubStatus,
      }));
      params.set('file', this.form.file);

      try {
        const resp = await fetch(`${urls.api}/import/subscribers`, {
          method: 'POST',
          body: params,
          headers: { Accept: 'application/json' },
        });

        const out = await resp.json().catch(() => ({}));
        if (!resp.ok) {
          u.toast(out.message || resp.statusText, 'danger');
          this.isProcessing = false;
          this.form.file = null;
          return;
        }

        // Reload so the server re-renders the page into the status view.
        u.reload({ message: i18n.t('import.importStarted') });
      } catch (err) {
        u.toast(err.message, 'danger');
        this.isProcessing = false;
        this.form.file = null;
      }
    },

    // Stop a running import or clear a finished one.
    async stopImport() {
      this.isProcessing = true;
      try {
        await fetch(`${urls.api}/import/subscribers`, {
          method: 'DELETE',
          headers: { Accept: 'application/json' },
        });
      } catch (err) { }

      // After clearing the server state, reload the page.
      if (this.isDone) {
        u.reload();
      } else {
        this.pollStatus();
      }
    },

    // ===============
    // Private functions.
    pollStatus() {
      clearInterval(this.pollID);

      const f = async () => {
        try {
          const resp = await fetch(`${urls.api}/import/subscribers`, { headers: { Accept: 'application/json' } });
          const out = await resp.json();
          this.isProcessing = false;
          this.status = out.data;
          await this.getLogs();

          if (!this.isRunning) {
            clearInterval(this.pollID);
          }
        } catch (err) {
          this.isProcessing = false;
          this.status = { status: 'none' };
          clearInterval(this.pollID);
        }
      };

      this.pollID = setInterval(f, 500);
      f();
    },

    async getLogs() {
      const resp = await fetch(`${urls.api}/import/subscribers/logs`, { headers: { Accept: 'application/json' } });
      const out = await resp.json();
      this.logLines = (out.data || '').split('\n')
        .filter((l) => l)
        .map((l) => this._splitLine(l.replace(/\s+importer\.go:\d+:\s*/, ' *: ')));

      this.$nextTick(() => {
        if (this.$refs.logLines) {
          this.$refs.logLines.scrollTop = this.$refs.logLines.scrollHeight;
        }
      });
    },

    _splitLine(l) {
      const parts = l.split(reLine);
      if (parts.length !== 5) {
        return { timestamp: '', file: '', message: l };
      }
      return { timestamp: parts[1], file: parts[2], message: parts[3] };
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('importView', component);
}, { once: true });
