import {
  api,
  urls,
  i18n,
  ListTag,
} from '../main.js';
import * as u from '../utils.js';

function component(sub = null) {
  // The current view's filters are injected by the template.
  const filters = window._filters || {};

  // Init a fresh form payload.
  const makeForm = () => ({
    id: null,
    uuid: '',
    email: '',
    name: '',
    status: 'enabled',
    lists: [],
    preconfirm: false,
    strAttribs: '{}',
  });
  const makeFormFromSub = (s) => ({
    ...makeForm(),
    ...s,
    lists: (Array.isArray(s.lists) ? s.lists : []).map((l) => new ListTag(l)),
    strAttribs: JSON.stringify(s.attribs || {}, null, 4),
  });

  return {
    filters,
    query: filters.query || '',
    isSqlOpen: false,
    get advanced() {
      return this.query.trim() !== '';
    },

    form: sub ? makeFormFromSub(sub) : makeForm(),

    subscriptions: sub && Array.isArray(sub.lists) ? sub.lists : [],

    // Manage-lists box state (subscriber edit page).
    manage: {
      action: 'add', lists: [], preconfirm: false,
    },

    // Bulk manage-lists dialog state.
    bulk: {
      action: 'add', lists: [], preconfirm: false, allSelected: false, selected: [],
    },

    // Bounces tab: controls the per-row meta JSON toggle (the rows are SSR-rendered).
    visibleMeta: {},

    // ===============
    // New / edit form.
    onOpenNew() {
      this.form = makeForm();
      this.$refs.dialog.showModal();
    },

    onClose() {
      this.$refs.dialog.close();
    },

    onDialogClose() {
      this.form = makeForm();
    },

    async onSubmitNew() {
      const payload = this._payload();
      if (!payload) {
        return;
      }
      const data = await api('subscribers.save', '/subscribers', 'POST', payload);
      u.reload({ message: i18n.ts('globals.messages.created', { name: data.email }) });
    },

    async onSubmitUpdate() {
      // PATCH the profile. Subscriptions (lists) are managed separately via the manage-lists dialog.
      const attribs = this._validateAttribs(this.form.strAttribs);
      if (attribs === null) {
        return;
      }
      const data = await api('subscribers.save', `/subscribers/${this.form.id}`, 'PATCH', {
        email: this.form.email,
        name: this.form.name,
        status: this.form.status,
        attribs,
      });

      u.reload({ message: i18n.ts('globals.messages.updated', { name: data.email }) });
    },

    onOpenManage() {
      this.manage = { action: 'add', lists: [], preconfirm: false };
      this.$refs.manageDialog.showModal();
    },

    // Apply the manage-lists action (add / remove / unsubscribe) to the subscriber for
    // the picked lists.
    async onManageLists() {
      const lists = this.manage.lists.map((l) => l.id);
      if (lists.length === 0) {
        return;
      }

      const data = {
        ids: [this.form.id],
        action: this.manage.action,
        target_list_ids: lists,
      };
      if (this.manage.preconfirm) {
        data.status = 'confirmed';
      }

      await api('subscribers', '/subscribers/lists', 'PUT', data);
      u.reload({ message: i18n.t('subscribers.listChangeApplied') });
    },

    async onDeleteSubscriber(id, email) {
      if (!(await u.confirm(i18n.ts('subscribers.confirmDelete', { num: 1 })))) {
        return;
      }

      await api('subscribers', `/subscribers/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name: email }) });
    },

    async onSendOptin() {
      if (!this.hasOptinSub) {
        return;
      }
      if (!(await u.confirm())) {
        return;
      }

      await api('subscribers', `/subscribers/${this.form.id}/optin`, 'POST');
      u.toast(i18n.t('subscribers.sentOptinConfirm'), 'success');
    },

    // ===============
    // Per-subscription actions in the Subscriptions tab.
    async _manageSub(action, listID, status) {
      if (!(await u.confirm())) {
        return;
      }

      await api('subscribers', '/subscribers/lists', 'PUT', {
        ids: [this.form.id],
        action,
        target_list_ids: [listID],
        status: status || '',
      });

      u.reload({ message: i18n.t('subscribers.listChangeApplied') });
    },

    onConfirmSub(id) {
      return this._manageSub('add', id, 'confirmed');
    },

    onResubscribe(id) {
      return this._manageSub('add', id, 'unconfirmed');
    },

    onUnsubscribe(id) {
      return this._manageSub('unsubscribe', id);
    },

    onDeleteSub(id) {
      return this._manageSub('remove', id);
    },

    // ===============
    // List selector helpers.
    get hasOptinList() {
      return this.form.lists.some((l) => l.optin === 'double');
    },

    get manageHasOptinList() {
      return this.manage.lists.some((l) => l.optin === 'double');
    },

    get hasOptinSub() {
      return this.subscriptions.some((l) => l.optin === 'double');
    },

    get bulkHasOptinList() {
      return this.bulk.lists.some((l) => l.optin === 'double');
    },

    // ===============
    // Bulk actions.
    onExport(allSelected, selected, total) {
      const num = (!allSelected && selected.length > 0) ? selected.length : total;
      u.confirm(i18n.ts('subscribers.confirmExport', { num })).then((ok) => {
        if (!ok) {
          return;
        }

        const q = new URLSearchParams();
        if (this.filters.search) {
          q.append('search', this.filters.search);
        } else if (this.filters.query) {
          q.append('query', this.filters.query);
        }
        if (this.filters.listID) {
          q.append('list_id', this.filters.listID);
        }
        if (this.filters.subStatus) {
          q.append('subscription_status', this.filters.subStatus);
        }
        if (!allSelected && selected.length > 0) {
          selected.forEach((id) => q.append('id', id));
        }

        window.location.href = `${urls.api}/subscribers/export?${q.toString()}`;
      });
    },

    async onDeleteSelected(allSelected, selected, total) {
      const num = allSelected ? total : selected.length;
      if (!(await u.confirm(i18n.ts('subscribers.confirmDelete', { num })))) {
        return;
      }

      if (!allSelected) {
        const q = new URLSearchParams();
        selected.forEach((id) => q.append('id', id));
        await api('subscribers', `/subscribers?${q.toString()}`, 'DELETE');
      } else {
        await api('subscribers', '/subscribers/query/delete', 'POST', this._queryBody());
      }

      u.reload({ message: i18n.ts('subscribers.subscribersDeleted', { num }) });
    },

    async onBlocklistSelected(allSelected, selected, total) {
      const num = allSelected ? total : selected.length;
      if (!(await u.confirm(i18n.ts('subscribers.confirmBlocklist', { num })))) {
        return;
      }

      if (!allSelected) {
        await api('subscribers', '/subscribers/blocklist', 'PUT', { ids: selected.map(Number) });
      } else {
        await api('subscribers', '/subscribers/query/blocklist', 'PUT', this._queryBody());
      }

      u.reload();
    },

    onOpenManageLists(allSelected, selected) {
      this.bulk = {
        action: 'add',
        lists: [],
        preconfirm: false,
        allSelected,
        selected: selected.map(Number),
      };
      this.$refs.listDialog.showModal();
    },

    async onSubmitManageLists() {
      const lists = this.bulk.lists.map((l) => l.id);
      if (lists.length === 0) {
        return;
      }

      const data = { action: this.bulk.action, target_list_ids: lists };
      if (this.bulk.preconfirm) {
        data.status = 'confirmed';
      }

      if (!this.bulk.allSelected) {
        data.ids = this.bulk.selected;
        await api('subscribers', '/subscribers/lists', 'PUT', data);
      } else {
        Object.assign(data, this._queryBody());
        await api('subscribers', '/subscribers/query/lists', 'PUT', data);
      }

      this.$refs.listDialog.close();
      u.reload({ message: i18n.t('subscribers.listChangeApplied') });
    },

    // ===============
    // Bounces tab.
    toggleMeta(id) {
      this.visibleMeta[id] = !this.visibleMeta[id];
    },

    async onDeleteBounces() {
      if (!(await u.confirm())) {
        return;
      }
      await api('subscribers', `/subscribers/${this.form.id}/bounces`, 'DELETE');
      u.reload({ message: i18n.t('globals.messages.done') });
    },

    // ===============
    // Private functions.
    // Body for the "by query" bulk endpoints.
    _queryBody() {
      const search = this.filters.search || '';
      const query = this.filters.query || '';
      return {
        all: query.trim() === '' && search.trim() === '',
        search,
        query,
        list_ids: this.filters.listID ? [this.filters.listID] : null,
        subscription_status: this.filters.subStatus || null,
      };
    },

    _payload() {
      const attribs = this._validateAttribs(this.form.strAttribs);
      if (attribs === null) {
        return null;
      }

      return {
        email: this.form.email,
        name: this.form.name,
        status: this.form.status,
        attribs,
        preconfirm_subscriptions: this.form.preconfirm,
        lists: this.form.lists.map((l) => l.id),
      };
    },

    _validateAttribs(str) {
      if (!str || !str.trim()) {
        return {};
      }
      try {
        const a = JSON.parse(str);
        if (Array.isArray(a)) {
          u.toast(i18n.t('subscribers.invalidJSON'), 'danger');
          return null;
        }
        return a;
      } catch (e) {
        u.toast(`${i18n.t('subscribers.invalidJSON')}: ${e.toString()}`, 'danger');
        return null;
      }
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('subscribersView', component);
}, { once: true });
