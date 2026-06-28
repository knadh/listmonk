import {
  api,
  urls,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

function component(sub = null) {
  // All lists (for the selector) and the current view's filters are injected by the template.
  const allLists = window._lists || [];
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
    lists: Array.isArray(s.lists) ? s.lists : [],
    strAttribs: JSON.stringify(s.attribs || {}, null, 4),
  });

  return {
    allLists,
    filters,
    advanced: Boolean(filters.query),
    listQuery: '',
    form: sub ? makeFormFromSub(sub) : makeForm(),

    // Bulk manage-lists dialog state.
    bulk: {
      action: 'add', lists: [], preconfirm: false, allSelected: false, selected: [],
    },

    // Edit-page tab data.
    bounces: [],
    visibleMeta: {},
    activity: { campaign_views: [], link_clicks: [] },
    activityLoaded: false,

    init() {
      // On the edit page, eagerly load the subscriber's bounces.
      if (this.form.id && window._canGetBounces) {
        this._loadBounces();
      }
    },

    // ===============
    // New / edit form.
    onOpenNew() {
      this.form = makeForm();
      this.listQuery = '';
      this.$refs.dialog.showModal();
    },

    onClose() {
      this.$refs.dialog.close();
    },

    onDialogClose() {
      this.form = makeForm();
      this.listQuery = '';
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
      const payload = this._payload();
      if (!payload) {
        return;
      }
      const data = await api('subscribers.save', `/subscribers/${this.form.id}`, 'PUT', payload);
      u.reload({ message: i18n.ts('globals.messages.updated', { name: data.email }) });
    },

    async onDeleteSubscriber(id, email) {
      if (!(await u.confirm(i18n.ts('subscribers.confirmDelete', { num: 1 })))) {
        return;
      }
      await api('subscribers', `/subscribers/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name: email }) });
    },

    async onSendOptin() {
      if (!this.hasOptinList) {
        return;
      }
      await api('subscribers', `/subscribers/${this.form.id}/optin`, 'POST');
      u.toast(i18n.t('subscribers.sentOptinConfirm'), 'success');
    },

    // ===============
    // List selector (bound to form.lists / bulk.lists).
    addList(model, ev) {
      const name = (ev && ev.target ? ev.target.value : '').trim();
      this.listQuery = '';
      if (ev && ev.target) {
        ev.target.value = '';
      }
      if (!name) {
        return;
      }

      const l = this.allLists.find((x) => x.name === name);
      const arr = this._resolve(model);
      if (l && !arr.some((x) => x.id === l.id)) {
        arr.push({
          id: l.id, name: l.name, type: l.type, optin: l.optin,
        });
      }
    },

    removeList(model, id) {
      const arr = this._resolve(model);
      const i = arr.findIndex((x) => x.id === id);
      if (i > -1) {
        arr.splice(i, 1);
      }
    },

    availableLists(selected) {
      const ids = new Set((selected || []).map((l) => l.id));
      const q = this.listQuery.toLowerCase();
      return this.allLists.filter((l) => !ids.has(l.id) && l.name.toLowerCase().includes(q));
    },

    get hasOptinList() {
      return this.form.lists.some((l) => l.optin === 'double');
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
      this.listQuery = '';
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
    // Edit-page tabs.
    async loadActivity() {
      if (this.activityLoaded) {
        return;
      }
      this.activityLoaded = true;
      this.activity = await api('subscribers.activity', `/subscribers/${this.form.id}/activity`, 'GET');
    },

    get totalViews() {
      return (this.activity.campaign_views || []).reduce((sum, v) => sum + (v.view_count || 0), 0);
    },

    get totalClicks() {
      return (this.activity.link_clicks || []).reduce((sum, c) => sum + (c.click_count || 0), 0);
    },

    toggleMeta(id) {
      this.visibleMeta[id] = !this.visibleMeta[id];
    },

    async onDeleteBounces() {
      if (!(await u.confirm())) {
        return;
      }
      await api('subscribers', `/subscribers/${this.form.id}/bounces`, 'DELETE');
      this._loadBounces();
    },

    // ===============
    // Labels / formatting.
    subStatusLabel(s) {
      return s ? i18n.t(`subscribers.status.${s}`) : '';
    },

    optinLabel(o) {
      return i18n.t(`lists.optins.${o}`);
    },

    fmtDate(s) {
      return s ? new Date(s).toLocaleString() : '';
    },

    // ===============
    // Private functions.
    async _loadBounces() {
      this.bounces = (await api('subscribers', `/subscribers/${this.form.id}/bounces`, 'GET')) || [];
    },

    // Resolve a dotted path (eg: "form.lists") to a live reference on this component.
    _resolve(path) {
      return path.split('.').reduce((o, k) => o[k], this);
    },

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
