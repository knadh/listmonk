import {
  api,
  config,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

function component(list = null) {
  // Init a fresh form payload.
  const makeForm = () => ({
    id: null,
    uuid: '',
    name: '',
    type: 'private',
    optin: 'single',
    status: 'active',
    tags: [],
    description: '',
  });
  const makeFormFromList = (list) => ({
    ...makeForm(),
    ...list,
    tags: Array.isArray(list.tags) ? list.tags : [],
  });

  return {
    form: list ? makeFormFromList(list) : makeForm(),

    // ===============
    // Event handlers.
    onOpenNew() {
      this._resetForm();
      this._openDialog();
    },

    onClose() {
      this.$refs.dialog.close();
    },

    onDialogClose() {
      this._resetForm();
    },

    async onSubmitNew() {
      const data = await api('lists.save', '/lists', 'POST', this._payload());
      u.reload({ message: i18n.ts('globals.messages.created', { name: data.name }) });
    },

    async onSubmitUpdate() {
      const data = await api('lists.save', `/lists/${this.form.id}`, 'PUT', this._payload());
      this.form = makeFormFromList(data);
      u.reload({ message: i18n.ts('globals.messages.updated', { name: data.name }) });
    },

    async onDeleteList(id, name) {
      if (!(await u.confirm(i18n.t('lists.confirmDelete')))) {
        return;
      }
      await api('lists', `/lists/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name }) });
    },

    async onDeleteSelected(count, params) {
      if (count === 0 || !(await u.confirm(i18n.ts('globals.messages.confirmDelete', {
        num: count,
        name: i18n.tc('globals.terms.list', count).toLowerCase(),
      })))) {
        return;
      }

      await api('lists', `/lists?${params}`, 'DELETE');
      u.reload({
        message: i18n.ts('globals.messages.deletedCount', {
          num: count,
          name: i18n.tc('globals.terms.list', count),
        })
      });
    },

    async onCreateOptinCampaign(id, name) {
      if (!(await u.confirm())) {
        return;
      }

      const data = await api('lists', '/campaigns', 'POST', {
        name: i18n.ts('lists.optinTo', { name }),
        subject: i18n.ts('lists.confirmSub', { name }),
        lists: [id],
        from_email: config.from_email,
        content_type: 'richtext',
        messenger: 'email',
        type: 'optin',
      });
      window.location.href = `/admin/campaigns/${data.id}#content`;
    },

    // ===============
    // Public functions.
    get isEditing() {
      return this.form.id !== null;
    },

    get dialogTitle() {
      return this.isEditing ? this.form.name : i18n.t('lists.newList');
    },

    get isArchived() {
      return this.form.status === 'archived';
    },

    set isArchived(value) {
      this.form.status = value ? 'archived' : 'active';
    },

    // ===============
    // Private functions.
    _resetForm() {
      this.form = makeForm();
    },

    _openDialog() {
      this.$refs.dialog.showModal();
      this.$nextTick(() => this.$refs.name.focus());
    },

    _payload() {
      return {
        name: this.form.name,
        type: this.form.type,
        optin: this.form.optin,
        status: this.form.status,
        tags: this.form.tags,
        description: this.form.description || '',
      };
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('listsView', component);
}, { once: true });
