import {
  api,
  isLoading,
  i18n,
  urls,
} from '../main.js';
import * as u from '../utils.js';

// Returns a 'YYYY-MM-DD' string for (today - n days), for <input type="date"> defaults.
function daysAgo(n) {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toISOString().slice(0, 10);
}

// Converts a 'YYYY-MM-DD' date input value to an RFC3339 timestamp.
function toRFC3339(date) {
  return new Date(date).toISOString();
}

function component() {
  return {
    isLoading,

    subscriberType: 'orphan',
    subscriptionType: 'optin',
    analyticsType: 'all',
    exportType: 'views',
    subscriptionDate: daysAgo(7),
    analyticsDate: daysAgo(7),
    exportDate: daysAgo(30),

    dbSettings: {
      vacuum: false,
      vacuum_cron_interval: '0 2 * * *',
      ...(window._dbSettings || {}),
    },
    isSaving: false,

    get exportURL() {
      if (!this.exportDate) {
        return '#';
      }
      const since = encodeURIComponent(toRFC3339(this.exportDate));
      return `${urls.api}/maintenance/analytics/${this.exportType}/export?since=${since}`;
    },

    // ===============
    // Event handlers.
    async deleteSubscribers() {
      if (!(await u.confirm())) {
        return;
      }

      const data = await api('maintenance', `/maintenance/subscribers/${this.subscriberType}`, 'DELETE');
      u.toast(i18n.ts('globals.messages.deletedCount', {
        name: i18n.tc('globals.terms.subscribers', 2),
        num: data.count,
      }));
    },

    async deleteSubscriptions() {
      if (!(await u.confirm())) {
        return;
      }

      const since = encodeURIComponent(toRFC3339(this.subscriptionDate));
      const data = await api('maintenance', `/maintenance/subscriptions/unconfirmed?before_date=${since}`, 'DELETE');
      u.toast(i18n.ts('globals.messages.deletedCount', {
        name: i18n.tc('globals.terms.subscriptions', 2),
        num: data.count,
      }));
    },

    async deleteAnalytics() {
      if (!(await u.confirm())) {
        return;
      }

      const since = encodeURIComponent(toRFC3339(this.analyticsDate));
      await api('maintenance', `/maintenance/analytics/${this.analyticsType}?before_date=${since}`, 'DELETE');
      u.toast(i18n.t('globals.messages.done'));
    },

    async onUpdateDBSettings() {
      this.isSaving = true;
      try {
        await api('settings', '/settings/maintenance.db', 'PUT', this.dbSettings);
        await this._awaitRestart();
      } finally {
        this.isSaving = false;
      }
    },

    // Poll the health endpoint until the app is back up after a settings-triggered
    // restart, then reload the page.
    _awaitRestart() {
      u.toast(i18n.ts('globals.messages.updated', { name: i18n.t('settings.title') }));
      return new Promise((resolve) => {
        const poll = setInterval(() => {
          fetch(`${urls.api}/health`).then((r) => {
            if (r.ok) {
              clearInterval(poll);
              resolve();
            }
          }).catch(() => { });
        }, 500);
      });
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('maintenanceView', component);
}, { once: true });
