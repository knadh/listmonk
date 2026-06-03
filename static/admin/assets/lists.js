import {
  api,
  config,
  i18n,
  init as initMain,
} from './main.js';
import * as u from './utils.js';

function listsPage({ query = '', total = 0 } = {}) {
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

  return {
    query,
    total,
    allSelected: false,
    selected: [],
    form: makeForm(),
    tagsText: '',
    isEditing: false,
    i18nReady: false,

    async init() {
      await initMain();
      this.i18nReady = true;
    },

    // ===============
    // Event handlers.
    onCheck() {
      this.allSelected = false;
      this.selected = u.checkedValues('input[name="id"]');
    },

    onToggleVisible(event) {
      document.querySelectorAll('input[name="id"]').forEach((el) => {
        el.checked = event.target.checked;
      });
      this.onCheck();
    },

    onSelectAllQuery() {
      this.allSelected = true;
      this.selected = [];
    },

    onOpenNew() {
      this._resetForm();
      this._openDialog();
    },

    onOpenEdit(list) {
      this.form = {
        ...makeForm(),
        ...list,
        tags: Array.isArray(list.tags) ? list.tags : [],
      };
      this.tagsText = this.form.tags.join(', ');
      this.isEditing = true;
      this._openDialog();
    },

    onClose() {
      this.$refs.dialog.close();
    },

    onDialogClose() {
      this._resetForm();
    },

    async onSubmit() {
      const uri = this.isEditing ? `/lists/${this.form.id}` : '/lists';
      const method = this.isEditing ? 'PUT' : 'POST';
      const data = await api('lists.save', uri, method, this._payload());
      u.toast(i18n.ts(
        this.isEditing ? 'globals.messages.updated' : 'globals.messages.created',
        { name: data.name },
      ));
      window.location.reload();
    },

    async onDeleteList(list) {
      if (!(await u.confirm(i18n.t('lists.confirmDelete')))) {
        return;
      }
      await api('lists.delete', `/lists/${list.id}`, 'DELETE');
      u.toast(i18n.ts('globals.messages.deleted', { name: list.name }));
      window.location.reload();
    },

    async onDeleteSelected() {
      const count = this.numSelected;
      if (count === 0 || !(await u.confirm(i18n.ts('globals.messages.confirmDelete', {
        num: count,
        name: i18n.t('globals.terms.list').toLowerCase(),
      })))) {
        return;
      }

      const params = new URLSearchParams();
      if (this.allSelected) {
        params.set('query', this.query || '');
        params.set('all', 'true');
      } else {
        this.selected.forEach((id) => params.append('id', id));
      }

      await api('lists.delete', `/lists?${params.toString()}`, 'DELETE');
      u.toast(i18n.ts('globals.messages.deletedCount', {
        num: count,
        name: i18n.t('globals.terms.list'),
      }));
      window.location.reload();
    },

    async onCreateOptinCampaign(list) {
      if (!(await u.confirm())) {
        return;
      }

      const data = await api('campaigns.create', '/campaigns', 'POST', {
        name: i18n.ts('lists.optinTo', { name: list.name }),
        subject: i18n.ts('lists.confirmSub', { name: list.name }),
        lists: [list.id],
        from_email: config.from_email,
        content_type: 'richtext',
        messenger: 'email',
        type: 'optin',
      });
      window.location.href = `/admin/campaigns/${data.id}#content`;
    },

    // ===============
    // Public functions.
    get isArchived() {
      return this.form.status === 'archived';
    },

    set isArchived(value) {
      this.form.status = value ? 'archived' : 'active';
    },

    get numSelected() {
      return this.allSelected ? this.total : this.selected.length;
    },

    get numSelectedLabel() {
      return i18n.ts('globals.messages.numSelected', { num: this.numSelected });
    },

    get selectAllLabel() {
      return i18n.ts('globals.messages.selectAll', { num: this.total });
    },

    // ===============
    // Private functions.
    _resetForm() {
      this.form = makeForm();
      this.tagsText = '';
      this.isEditing = false;
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
        tags: this.tagsText.split(',').map((tag) => tag.trim()).filter(Boolean),
        description: this.form.description || '',
      };
    },
  };
}

window.listsPage = listsPage;
