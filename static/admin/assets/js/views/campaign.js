import { convert, htmlToText } from '../content.js';
import {
  api,
  config,
  i18n,
  ListTag,
  urls,
} from '../main.js';
import * as u from '../utils.js';

// Media tag for <ot-taginput> attachments.
class MediaTag {
  constructor(m) { Object.assign(this, m); }

  toString() { return this.filename; }
}

// Convert an ISO datetime to a value for <input type="datetime-local"> in local time.
function toLocalInput(iso) {
  if (!iso) {
    return '';
  }

  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return '';
  }

  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function component(camp, sel) {
  const c = camp || {};
  const isNew = !c.id;

  // Initial lists. For a new campaign, resolve optional `?list_id` query params.
  let lists = Array.isArray(c.lists) ? c.lists.slice() : [];
  if (isNew && Array.isArray(sel) && sel.length > 0) {
    lists = (window._lists || []).filter((l) => sel.includes(l.id));
  }

  return {
    isNew,
    isHeadersVisible: Array.isArray(c.headers) && c.headers.length > 0,
    contentReady: false,
    editorReady: false,
    visualTemplateId: null,
    _mediaTarget: null,
    _saved: false,

    preview: {
      title: c.name || '', url: '', loading: true, templateID: null, contentType: '', archiveMeta: '', body: '',
    },

    form: {
      id: c.id || null,
      uuid: c.uuid || '',
      status: c.status || 'draft',
      type: c.type || 'regular',
      name: c.name || '',
      subject: c.subject || '',
      from_email: c.from_email || config.from_email || '',
      messenger: c.messenger || 'email',
      content_type: c.content_type || 'richtext',
      tags: Array.isArray(c.tags) ? c.tags : [],
      lists: lists.map((l) => new ListTag(l)),
      template_id: c.template_id || null,
      body: c.body || '',
      body_source: c.body_source || null,
      altbody: c.altbody === undefined ? null : c.altbody,
      media: (Array.isArray(c.media) ? c.media : []).map((m) => new MediaTag(m.id ? m : { ...m, filename: `❌ ${m.filename}` })),
      headersStr: JSON.stringify(c.headers || [], null, 4),
      attribsStr: c.attribs ? JSON.stringify(c.attribs, null, 4) : '{}',
      archive: !!c.archive,
      archive_slug: c.archive_slug || '',
      archive_template_id: c.archive_template_id || null,
      archiveMetaStr: c.archive_meta ? JSON.stringify(c.archive_meta, null, 4) : '{}',
      sendLater: !!c.send_at,
      sendAtLocal: toLocalInput(c.send_at),
      testEmails: [],
    },

    init() {
      // Default the content + archive templates if none are set.
      const def = this.templatesFor('campaign').find((t) => t.is_default) || this.templatesFor('campaign')[0];
      if (def) {
        if (!this.form.template_id) {
          this.form.template_id = def.id;
        }
        if (!this.form.archive_template_id) {
          this.form.archive_template_id = def.id;
        }
      }

      // Mount content editor plugins only when the 'content' tab is active.
      try {
        if (new URLSearchParams(window.location.hash.slice(1)).get('tab') === 'content') {
          this.contentReady = true;
        }
      } catch (e) { }
      this.$root.addEventListener('ot-tab-change', (e) => {
        if (e.detail?.tab?.id === 'content') {
          this.contentReady = true;
        }
      });

      // Listen for the media picker selections pushed from the dialog.
      window.addEventListener('message', (e) => {
        if (e.origin !== window.location.origin || e.source !== this.$refs.mediaFrame?.contentWindow) {
          return;
        }

        if (e.data?.type === 'media-select') {
          this.onPickMedia(e.data.media);
        }
      });
    },

    // ===============
    // Status-based computed getters.
    get canEdit() {
      return this.isNew || ['draft', 'scheduled', 'paused'].includes(this.form.status);
    },

    get canStart() {
      return ['draft', 'paused'].includes(this.form.status) && !this.form.sendLater;
    },

    get canSchedule() {
      return ['draft', 'paused'].includes(this.form.status) && this.form.sendLater && !!this.form.sendAtLocal;
    },

    get canUnSchedule() {
      return this.form.status === 'scheduled';
    },

    get canArchive() {
      return this.form.status !== 'cancelled' && this.form.type !== 'optin';
    },

    get showEditorSpinner() {
      return this.contentReady && this.form.content_type !== 'plain' && !this.editorReady;
    },

    templatesFor(type) {
      return (window._templates || []).filter((t) => t.type === type);
    },

    onToggleSendLater() {
      // Prefill a +7 day schedule if there's no existing value.
      if (this.form.sendLater && !this.form.sendAtLocal) {
        const d = new Date();
        d.setDate(d.getDate() + 7);
        d.setHours(0, 0, 0, 0);
        this.form.sendAtLocal = toLocalInput(d.toISOString());
      }
    },

    // ===============
    // Content type switching (converts the body between formats).
    async onContentTypeChange(e) {
      const to = e.target.value;
      const from = this.form.content_type;
      if (to === from) {
        return;
      }

      // Confirm before converting a non-empty body.
      if (this.form.body && this.form.body.trim()
        && !(await u.confirm(i18n.t('campaigns.confirmSwitchFormat')))) {
        e.target.value = from;
        return;
      }

      const { body, bodySource } = convert(this.form.body, from, to);
      if (to === 'visual' || from === 'visual') {
        this.form.template_id = null;
      }

      // The new editor will remount. Hide it behind the spinner until it's ready.
      this.editorReady = false;
      this.form.content_type = to;
      this.form.body = body;
      this.form.body_source = bodySource;
      this.$nextTick(() => this._syncEditor());
    },

    // Push the converted body into the new editor element.
    _syncEditor() {
      const el = this.$root.querySelector('.editor-body richtext-editor, .editor-body code-editor');
      if (el) {
        el.value = this.form.body;
      }

      if (this.form.content_type === 'visual' && this.form.body_source) {
        const ve = this.$root.querySelector('visual-editor');
        if (ve) {
          try { ve.render(JSON.parse(this.form.body_source)); } catch (err) { /* noop */ }
        }
      }
    },

    // ===============
    // Save / create / test / status.

    _payload() {
      let headers = [];
      const hs = (this.form.headersStr || '').trim();
      if (hs && hs !== '[]') {
        try { headers = JSON.parse(hs); } catch (e) { u.toast(e.toString(), 'danger'); return null; }
      }

      let attribs = null;
      const as = (this.form.attribsStr || '').trim();
      if (as) {
        try { attribs = JSON.parse(as); } catch (e) { u.toast(`${i18n.t('subscribers.invalidJSON')}: ${e}`, 'danger'); return null; }
      }

      let archiveMeta = {};
      if (this.form.archive && this.form.archiveMetaStr) {
        try { archiveMeta = JSON.parse(this.form.archiveMetaStr); } catch (e) { u.toast(e.toString(), 'danger'); return null; }
      }

      return {
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.from_email,
        messenger: this.form.messenger,
        type: 'regular',
        tags: this.form.tags,
        send_at: (this.form.sendLater && this.form.sendAtLocal) ? new Date(this.form.sendAtLocal).toISOString() : null,
        headers,
        attribs,
        template_id: this.form.template_id,
        content_type: this.form.content_type,
        body: this.form.body,
        body_source: this.form.body_source,
        altbody: this.form.content_type !== 'plain' ? this.form.altbody : null,
        archive: this.form.archive,
        archive_slug: this.form.archive_slug,
        archive_template_id: this.form.archive_template_id,
        archive_meta: archiveMeta,
        media: this.form.media.filter((m) => m.id).map((m) => m.id),
      };
    },

    async onCreate() {
      const p = this._payload();
      if (!p) {
        return;
      }

      const data = {
        archive_slug: this.form.subject,
        name: p.name,
        subject: p.subject,
        lists: p.lists,
        from_email: p.from_email,
        content_type: p.content_type,
        messenger: p.messenger,
        type: 'regular',
        tags: p.tags,
        send_at: p.send_at,
        headers: p.headers,
        attribs: p.attribs,
        media: p.media,
      };

      const d = await api('campaigns', '/campaigns', 'POST', data);
      this._saved = true;
      u.redirect(`${urls.admin}/campaigns/${d.id}#tab=content`, { message: i18n.ts('globals.messages.created', { name: d.name }) });
    },

    async onSave() {
      const p = this._payload();
      if (!p) {
        return;
      }

      const d = await api('campaigns', `/campaigns/${this.form.id}`, 'PUT', p);
      this._saved = true;
      this.form.archive_slug = d.archive_slug || this.form.archive_slug;
      u.toast(i18n.ts('globals.messages.updated', { name: d.name }));
    },

    // Global Ctrl/Cmd+S handler.
    onSaveShortcut() {
      if (this.isNew) {
        this.onCreate();
      } else if (this.canEdit) {
        this.onSave();
      } else if (this.canArchive) {
        this.onSaveArchive();
      }
    },

    async onStart() {
      if (!this.canStart && !this.canSchedule) {
        return;
      }
      if (!(await u.confirm())) {
        return;
      }

      const status = this.canStart ? 'running' : 'scheduled';
      await this.onSave();
      await api('campaigns', `/campaigns/${this.form.id}/status`, 'PUT', { status });
      u.redirect(`${urls.admin}/campaigns`, { message: i18n.ts('campaigns.statusChanged', { name: this.form.name, status }) });
    },

    async onUnschedule() {
      if (!(await u.confirm())) {
        return;
      }

      await api('campaigns', `/campaigns/${this.form.id}/status`, 'PUT', { status: 'draft' });
      u.reload();
    },

    async onTest() {
      if (this.isNew) {
        return;
      }

      const p = this._payload();
      if (!p) {
        return;
      }

      await api('campaigns', `/campaigns/${this.form.id}/test`, 'POST', {
        ...p,
        id: this.form.id,
        subscribers: this.form.testEmails,
      });
      u.toast(i18n.t('campaigns.testSent'));
    },

    async onSaveArchive() {
      let meta = {};
      try { meta = JSON.parse(this.form.archiveMetaStr || '{}'); } catch (e) { u.toast(e.toString(), 'danger'); return; }

      const d = await api('campaigns', `/campaigns/${this.form.id}/archive`, 'PUT', {
        archive: this.form.archive,
        archive_template_id: this.form.archive_template_id,
        archive_meta: meta,
        archive_slug: this.form.archive_slug,
      });
      this.form.archive_slug = d.archive_slug || this.form.archive_slug;

      u.toast(i18n.t('globals.messages.done'));
    },

    // ===============
    // Alt body / headers / archive meta helpers.
    onAddAltBody() {
      this.form.altbody = htmlToText(this.form.body || '');
    },

    async onRemoveAltBody() {
      if (await u.confirm()) {
        this.form.altbody = null;
      }
    },

    onFillArchiveMeta() {
      const s = `{"email": "email@domain.com", "name": "${i18n.t('globals.fields.name')}", "attribs": {}}`;
      this.form.archiveMetaStr = JSON.stringify(JSON.parse(s), null, 4);
    },

    async onImportVisualTpl() {
      if (!this.visualTemplateId) {
        return;
      }

      if (!(await u.confirm(i18n.t('campaigns.confirmOverwriteContent')))) {
        return;
      }

      const d = await api('templates', `/templates/${this.visualTemplateId}`);
      this.form.body = d.body;
      this.form.body_source = d.body_source;

      const ve = this.$root.querySelector('visual-editor');
      if (ve && d.body_source) {
        try { ve.render(JSON.parse(d.body_source)); } catch (err) { /* noop */ }
      }
    },

    // ===============
    // Preview via the reusable campaign-preview dialog.
    onPreviewContent() {
      this.preview.title = this.form.name;
      this.preview.loading = true;
      this.preview.url = `${urls.api}/campaigns/${this.form.id}/preview`;
      this.preview.templateID = this.form.template_id || '';
      this.preview.contentType = this.form.content_type;
      this.preview.archiveMeta = '';
      this.preview.body = this.form.body || '';
      this.$refs.previewDialog.showModal();
      this.$nextTick(() => this.$refs.previewForm.submit());
    },

    onPreviewArchive() {
      this.preview.title = this.form.name;
      this.preview.loading = true;
      this.preview.url = `${urls.api}/campaigns/${this.form.id}/preview/archive`;
      this.preview.templateID = this.form.archive_template_id || '';
      this.preview.contentType = this.form.content_type;
      this.preview.archiveMeta = this.form.archiveMetaStr;
      this.preview.body = '';
      this.$refs.previewDialog.showModal();
      this.$nextTick(() => this.$refs.previewForm.submit());
    },

    onPreviewClose() {
      this.$refs.previewFrame.removeAttribute('src');
      this.preview.loading = true;
    },

    // ===============
    // Media picker.
    _openMedia(target) {
      this._mediaTarget = target;
      this.$refs.mediaFrame.src = `${urls.admin}/campaigns/media/fragment?t=${Date.now()}`;
      this.$refs.mediaDialog.showModal();
    },

    onOpenMedia(target) {
      this._openMedia(target);
    },

    onEditorMedia(e) {
      this._openMedia(e.type === 'richtext-media' ? e.detail.cb : 'visual');
    },

    onMediaClose() {
      this._mediaTarget = null;
    },

    onPickMedia(m) {
      const t = this._mediaTarget;
      if (typeof t === 'function') {
        t(m.url);
      } else if (t === 'visual') {
        const ve = this.$root.querySelector('visual-editor');
        if (ve) {
          ve.insertMedia(m.url);
        }
      } else if (t === 'attach' && !this.form.media.some((x) => x.id === m.id)) {
        this.form.media.push(new MediaTag({
          id: m.id, filename: m.filename, url: m.url, thumb_url: m.thumb_url, created_at: m.created_at,
        }));
      }

      this.$refs.mediaDialog.close();
    },

    fileExt(filename) {
      const i = (filename || '').lastIndexOf('.');
      return i > 0 ? filename.slice(i + 1).toUpperCase() : '';
    },

    // Used for the attachment gallery captions to match the server-rendered media gallery.
    niceDate: u.niceDate,

    // Remove an attachment. It's persisted only when the campaign is saved.
    async onDeleteAttachment(m) {
      if (!(await u.confirm())) {
        return;
      }
      this.form.media = this.form.media.filter((x) => x !== m);
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('campaignView', component);
}, { once: true });
