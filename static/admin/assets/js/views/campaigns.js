import Alpine from 'alpinejs';
import {
  api,
  urls,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

function component() {
  return {
    // Preview dialog state (bound in the reusable campaign-preview partial).
    preview: {
      title: '', url: '', loading: true, templateID: null, contentType: '', archiveMeta: '', body: '',
    },

    // Running-campaign stats polling.
    _pollID: null,

    init() {
      if (this.$el.querySelector('tr.status-running')) {
        this._pollID = setTimeout(() => this._poll(), 0);
      }
    },

    destroy() {
      clearTimeout(this._pollID);
      this._pollID = null;
    },

    // ===============
    // Row actions.
    onPreview(id, name) {
      this.preview.title = name;
      this.preview.loading = true;
      this.$refs.previewDialog.showModal();

      // Direct GET-preview of the saved campaign body.
      this.$refs.previewFrame.src = `${urls.api}/campaigns/${id}/preview`;
    },

    onPreviewClose() {
      this.$refs.previewFrame.removeAttribute('src');
      this.preview.loading = true;
    },

    async onChangeStatus(id, name, status) {
      const msg = status === 'scheduled' ? i18n.t('campaigns.confirmSchedule') : null;
      if (!(await u.confirm(msg))) {
        return;
      }

      await api('campaigns', `/campaigns/${id}/status`, 'PUT', { status });
      u.reload({ message: i18n.ts('globals.messages.statusChanged', { name, status: i18n.t(`globals.terms.${status}`) }) });
    },

    async onClone(id, name) {
      const newName = window.prompt(i18n.t('globals.fields.name'), i18n.ts('globals.messages.copyOf', { name }));
      if (!newName || !newName.trim()) {
        return;
      }

      // Fetch the full campaign (with body) to clone.
      const c = await api('campaigns', `/campaigns/${id}`);
      const data = {
        name: newName,
        subject: c.subject,
        lists: (c.lists || []).map((l) => l.id),
        type: c.type,
        from_email: c.from_email,
        content_type: c.content_type,
        messenger: c.messenger,
        tags: c.tags,
        template_id: c.template_id,
        body: c.body,
        body_source: c.body_source,
        altbody: c.altbody,
        headers: c.headers,
        attribs: c.attribs,
        send_at: c.send_at,
        archive: c.archive,
        archive_template_id: c.archive_template_id,
        archive_meta: c.archive_meta,
        media: (c.media || []).map((m) => m.id),
      };
      if (c.archive) {
        data.archive_slug = `${newName.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${Date.now().toString().slice(-4)}`;
      }

      const d = await api('campaigns', '/campaigns', 'POST', data);
      u.redirect(`/admin/campaigns/${d.id}`, { message: i18n.ts('globals.messages.created', { name: d.name }) });
    },

    async onDelete(id, name) {
      if (!(await u.confirm(i18n.ts('globals.messages.confirmDeleteNamed', { name })))) {
        return;
      }
      await api('campaigns', `/campaigns/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name }) });
    },

    // ===============
    // Bulk actions (called from within the table() scope).
    async onDeleteSelected(count, params) {
      if (count === 0 || !(await u.confirm(i18n.ts('globals.messages.confirmDelete', {
        num: count,
        name: i18n.tc('globals.terms.campaign', count).toLowerCase(),
      })))) {
        return;
      }

      await api('campaigns', `/campaigns?${params}`, 'DELETE');
      u.reload({
        message: i18n.ts('globals.messages.deletedCount', {
          num: count,
          name: i18n.tc('globals.terms.campaign', count),
        }),
      });
    },

    // ===============
    // Live stats polling for running campaigns.
    async _poll() {
      try {
        const resp = await fetch(`${urls.api}/campaigns/running/stats`, {
          headers: { Accept: 'application/json' },
          signal: AbortSignal.timeout(10000),
        });
        if (!resp.ok) {
          if (resp.status === 401 || resp.status === 403) this.destroy();
          return;
        }

        const { data } = await resp.json();
        if (this._pollID === null || !Array.isArray(data)) return;

        for (const row of this.$el.querySelectorAll('tr.status-running[data-campaign-id]')) {
          const stats = data.find((c) => c.id === Number(row.dataset.campaignId));
          if (!stats) {
            this.destroy();
            u.reload();
            return;
          }
          this._updateStats(stats);
        }
      } catch (e) {
        // Retry transient network, timeout, and JSON errors.
      } finally {
        // Schedule after completion so requests cannot overlap.
        if (this._pollID !== null) this._pollID = setTimeout(() => this._poll(), 1000);
      }
    },

    _updateStats(c) {
      const row = document.querySelector(`tr[data-campaign-id="${c.id}"]`);
      if (!row) {
        return;
      }

      const set = (name, val) => {
        const el = row.querySelector(`[data-stat="${name}"]`);
        if (el) el.textContent = val;
      };
      set('sent', u.formatNumber(c.sent));
      const sentCount = row.querySelector('[data-sent-count]');
      if (sentCount) {
        sentCount.hidden = false;
      }
      set('to_send', u.formatNumber(c.to_send));
      const sendTotal = row.querySelector('[data-send-total]');
      if (sendTotal) {
        sendTotal.hidden = c.to_send === 0;
      }
      set('rate', Math.round(c.rate || 0));
      set('percent', `${c.to_send > 0 ? Math.floor((c.sent / c.to_send) * 100) : 0}%`);

      const prog = row.querySelector('[data-stat="progress"]');
      if (prog) {
        prog.value = c.sent;
        prog.max = c.to_send || 1;
      }
    },
  };
}

document.addEventListener('alpine:init', () => {
  Alpine.data('campaignsView', component);
}, { once: true });
