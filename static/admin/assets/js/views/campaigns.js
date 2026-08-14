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
    _hadRunning: false,

    init() {
      this._poll();
      this._pollID = setInterval(() => this._poll(), 1000);
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
      u.reload({ message: i18n.ts('campaigns.statusChanged', { name, status }) });
    },

    async onClone(id, name) {
      const newName = window.prompt(i18n.t('globals.fields.name'), i18n.ts('campaigns.copyOf', { name }));
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
      if (!(await u.confirm(i18n.ts('campaigns.confirmDelete', { name })))) {
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
      let resp;
      try {
        resp = await fetch(`${urls.api}/campaigns/running/stats`, { headers: { Accept: 'application/json' } });
      } catch (e) {
        clearInterval(this._pollID);
        return;
      }
      if (!resp.ok) {
        clearInterval(this._pollID);
        return;
      }

      const out = await resp.json().catch(() => ({}));
      const data = Array.isArray(out.data) ? out.data : [];

      // No running campaigns. Stop polling, and if some were running earlier,
      // reload to fetch fresh server-rendered stats and statuses.
      if (data.length === 0) {
        clearInterval(this._pollID);
        if (this._hadRunning) {
          u.reload();
        }
        return;
      }

      this._hadRunning = true;
      data.forEach((c) => this._updateStats(c));
    },

    _updateStats(c) {
      const row = document.querySelector(`tr[data-campaign-id="${c.id}"]`);
      if (!row) {
        return;
      }

      const set = (name, val) => {
        const el = row.querySelector(`[data-stat="${name}"]`);
        if (el) {
          el.textContent = val;
        }
      };
      set('sent', u.formatNumber(c.sent));
      set('to_send', u.formatNumber(c.to_send));
      set('rate', Math.round(c.rate || 0));

      const prog = row.querySelector('[data-stat="progress"]');
      if (prog) {
        prog.value = c.sent;
        prog.max = c.to_send;
      }
    },
  };
}

document.addEventListener('alpine:init', () => {
  Alpine.data('campaignsView', component);
}, { once: true });
