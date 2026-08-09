import {
  api,
  urls,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

// previewMixin provides the shared state + handlers for the template-preview dialog.
function previewMixin() {
  return {
    preview: {
      title: '', url: '', templateType: '', body: '', loading: true,
    },

    // Preview a saved template by id (GET into the iframe).
    onPreview(id, name) {
      this.preview = {
        title: name, url: `${urls.api}/templates/${id}/preview`, templateType: '', body: '', loading: true,
      };
      this.$refs.previewDialog.showModal();
      this.$nextTick(() => { this.$refs.previewFrame.src = this.preview.url; });
    },

    onPreviewClose() {
      this.$refs.previewFrame.removeAttribute('src');
      this.preview.loading = true;
    },
  };
}

// templatesView is the templates list page component.
function templatesView() {
  return {
    ...previewMixin(),

    // Clone a template by fetching its full body and creating a copy.
    async onClone(id, name) {
      const newName = window.prompt(i18n.t('globals.fields.name'), i18n.ts('campaigns.copyOf', { name }));
      if (!newName) {
        return;
      }

      const t = await api('templates', `/templates/${id}`);
      const data = await api('templates', '/templates', 'POST', {
        name: newName,
        type: t.type,
        subject: t.subject,
        body: t.body,
        body_source: t.body_source,
      });
      u.reload({ message: i18n.ts('globals.messages.created', { name: data.name }) });
    },

    async onSetDefault(id, name) {
      if (!(await u.confirm())) {
        return;
      }
      await api('templates', `/templates/${id}/default`, 'PUT');
      u.reload({ message: i18n.ts('globals.messages.created', { name }) });
    },

    async onDelete(id, name) {
      if (!(await u.confirm(i18n.t('globals.messages.confirm')))) {
        return;
      }
      await api('templates', `/templates/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name }) });
    },
  };
}

// templateForm is the add/edit template form page component.
function templateForm(tpl, isNew) {
  const t = tpl || {};

  return {
    ...previewMixin(),
    isNew,

    form: {
      id: t.id || null,
      name: t.name || '',
      type: t.type || 'campaign',
      subject: t.subject || '',
      body: t.body || '',
      body_source: t.body_source || null,
    },

    // Preview the current (unsaved) body by POSTing it to the preview endpoint.
    onPreviewBody() {
      this.preview = {
        title: this.form.name, url: `${urls.api}/templates/preview`, templateType: this.form.type, body: this.form.body || '', loading: true,
      };
      this.$refs.previewDialog.showModal();
      this.$nextTick(() => this.$refs.previewForm.submit());
    },

    onSaveShortcut() {
      this.onSubmit();
    },

    onSubmit() {
      if (this.isNew) {
        this.createTemplate();
      } else {
        this.updateTemplate();
      }
    },

    _payload() {
      return {
        name: this.form.name,
        type: this.form.type,
        subject: this.form.subject,
        body: this.form.body,
        body_source: this.form.body_source,
      };
    },

    async createTemplate() {
      const data = await api('templates', '/templates', 'POST', this._payload());
      u.redirect(`${urls.admin}/templates`, { message: i18n.ts('globals.messages.created', { name: data.name }) });
    },

    async updateTemplate() {
      const data = await api('templates', `/templates/${this.form.id}`, 'PUT', this._payload());
      u.reload({ message: i18n.ts('globals.messages.updated', { name: data.name }) });
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('templatesView', templatesView);
  window.Alpine.data('templateForm', templateForm);
}, { once: true });
