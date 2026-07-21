import {
  api,
  urls,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

// rolesView is the list page component (user or list roles).
function rolesView({ type = 'user' } = {}) {
  const path = type === 'list' ? '/roles/lists' : '/roles/users';

  return {
    // Clone a role by fetching its full definition and creating a copy.
    async onCloneRole(id, name) {
      const newName = window.prompt(i18n.t('globals.fields.name'), i18n.ts('campaigns.copyOf', { name }));
      if (!newName) {
        return;
      }

      const roles = await api('roles', path);
      const r = (roles || []).find((x) => x.id === id);
      if (!r) {
        return;
      }

      const payload = { name: newName };
      if (type === 'list') {
        payload.lists = (r.lists || []).map((l) => ({ id: l.id, permissions: l.permissions }));
      } else {
        payload.permissions = r.permissions;
      }

      await api('roles', path, 'POST', payload);
      u.reload({ message: i18n.ts('globals.messages.created', { name: newName }) });
    },

    async onDeleteRole(id, name) {
      if (!(await u.confirm(i18n.t('globals.messages.confirm')))) {
        return;
      }
      await api('roles', `/roles/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name }) });
    },
  };
}

// roleForm is the add/edit form page component (user or list roles).
function roleForm({ type = 'user', isEditing = false, canManage = false } = {}) {
  const role = window._role || {};
  const permGroups = window._permGroups || [];
  const allLists = window._lists || [];
  const path = type === 'list' ? '/roles/lists' : '/roles/users';

  const makeForm = () => ({
    id: null, name: '', permissions: [], lists: [], curList: null,
  });

  // The default set of permissions pre-selected for a new user role.
  const defaultUserPerms = () => permGroups.reduce((acc, g) => {
    if (g.group === 'admin' || g.group === 'users') {
      return acc;
    }
    g.permissions.forEach((p) => {
      if (p !== 'subscribers:sql_query' && !p.startsWith('lists:') && !p.startsWith('settings:')) {
        acc.push(p);
      }
    });
    return acc;
  }, []);

  const fromRole = () => {
    const f = { ...makeForm(), id: role.id, name: role.name || '' };
    if (type === 'list') {
      f.lists = (role.lists || []).map((l) => ({ id: l.id, name: l.name, permissions: [...(l.permissions || [])] }));
    } else {
      f.permissions = [...(role.permissions || [])];
    }
    return f;
  };

  return {
    // The primordial super admin role (id 1) or lacking manage permission disables the form.
    disabled: !canManage || (isEditing && role.id === 1),
    form: isEditing ? fromRole() : { ...makeForm(), permissions: type === 'user' ? defaultUserPerms() : [] },

    get isEditing() {
      return isEditing;
    },

    // Lists not yet added to the role.
    get filteredLists() {
      const chosen = new Set(this.form.lists.map((l) => l.id));
      return allLists.filter((l) => !chosen.has(l.id));
    },

    init() {
      if (type === 'list' && this.filteredLists.length > 0) {
        this.form.curList = this.filteredLists[0].id;
      }
    },

    onAddListPerm() {
      const l = allLists.find((x) => x.id === this.form.curList);
      if (!l) {
        return;
      }
      this.form.lists.push({ id: l.id, name: l.name, permissions: ['list:get', 'list:manage'] });
      this.form.curList = this.filteredLists.length > 0 ? this.filteredLists[0].id : null;
    },

    onDeleteListPerm(id) {
      this.form.lists = this.form.lists.filter((l) => l.id !== id);
      this.form.curList = this.filteredLists.length > 0 ? this.filteredLists[0].id : null;
    },

    onToggleSelect() {
      if (this.form.permissions.length > 0) {
        this.form.permissions = [];
      } else {
        this.form.permissions = permGroups.reduce((acc, g) => acc.concat(g.permissions), []);
      }
    },

    onSubmit() {
      if (this.disabled) {
        return;
      }
      if (this.isEditing) {
        this.updateRole();
      } else {
        this.createRole();
      }
    },

    _payload() {
      const payload = { name: this.form.name };
      if (type === 'list') {
        payload.lists = this.form.lists.map((l) => ({ id: l.id, permissions: l.permissions }));
      } else {
        payload.permissions = this.form.permissions;
      }
      return payload;
    },

    _listURL() {
      return type === 'list' ? `${urls.admin}/users/roles/lists` : `${urls.admin}/users/roles`;
    },

    async createRole() {
      const data = await api('roles.save', path, 'POST', this._payload());
      u.redirect(this._listURL(), { message: i18n.ts('globals.messages.created', { name: data.name }) });
    },

    async updateRole() {
      const data = await api('roles.save', `${path}/${this.form.id}`, 'PUT', this._payload());
      u.reload({ message: i18n.ts('globals.messages.updated', { name: data.name }) });
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('rolesView', rolesView);
  window.Alpine.data('roleForm', roleForm);
}, { once: true });
