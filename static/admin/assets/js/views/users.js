import {
  api,
  urls,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

// usersView is the users list page component.
function usersView() {
  return {
    // ===============
    // Table actions.
    async onDeleteUser(id, name) {
      if (!(await u.confirm(i18n.t('globals.messages.confirm')))) {
        return;
      }

      await api('users', `/users/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name }) });
    },

    async onDeleteSelected(count, params) {
      if (count === 0 || !(await u.confirm(i18n.ts('globals.messages.confirmDelete', {
        num: count,
        name: i18n.tc('globals.terms.user', count).toLowerCase(),
      })))) {
        return;
      }

      await api('users', `/users?${params}`, 'DELETE');
      u.reload({
        message: i18n.ts('globals.messages.deletedCount', {
          num: count,
          name: i18n.tc('globals.terms.user', count),
        }),
      });
    },
  };
}

// userForm is the add/edit form page component.
function userForm({ isEditing = false, canSave = false } = {}) {
  const user = window._user || {};
  const userRoles = window._userRoles || [];

  // Init a fresh form payload.
  const makeForm = () => ({
    id: null,
    username: '',
    name: '',
    email: '',
    password: '',
    password2: '',
    password_login: false,
    type: 'user',
    status: 'enabled',
    user_role_id: userRoles.length > 0 ? userRoles[0].id : null,
    list_role_id: '',
  });

  const fromUser = (us) => ({
    ...makeForm(),
    ...us,
    email: us.email || '',
    password: '',
    password2: '',
    user_role_id: us.user_role ? us.user_role.id : us.user_role_id,
    list_role_id: us.list_role ? us.list_role.id : '',
  });

  return {
    form: isEditing ? fromUser(user) : makeForm(),
    apiToken: null,
    createdName: '',

    get isEditing() {
      return isEditing;
    },

    onSubmit() {
      if (!canSave || !this._validate()) {
        return;
      }
      if (isEditing) {
        this.updateUser();
      } else {
        this.createUser();
      }
    },

    // Redirect to the users list once the one-time API token dialog is dismissed.
    onTokenDialogClose() {
      u.redirect(`${urls.admin}/users`, { message: i18n.ts('globals.messages.created', { name: this.createdName }) });
    },

    async createUser() {
      const data = await api('users.save', '/users', 'POST', this._payload());

      // For API users, show the one-time token in a dialog before redirecting.
      if (data.type === 'api') {
        this.apiToken = data.password;
        this.createdName = data.name;
        this.$refs.tokenDialog.showModal();
        return;
      }

      u.redirect(`${urls.admin}/users`, { message: i18n.ts('globals.messages.created', { name: data.name }) });
    },

    async updateUser() {
      const data = await api('users.save', `/users/${this.form.id}`, 'PUT', this._payload());
      u.reload({ message: i18n.ts('globals.messages.updated', { name: data.name }) });
    },

    // ===============
    // Private functions.
    _validate() {
      const f = this.form;
      const withPassword = f.type !== 'api' && f.password_login;
      if (withPassword && f.password && f.password !== f.password2) {
        u.toast(i18n.t('users.passwordMismatch'), 'danger');
        return false;
      }

      return true;
    },

    _payload() {
      const f = this.form;
      return {
        username: f.username,
        name: f.name,
        email: f.email,
        type: f.type,
        status: f.status,
        password_login: f.type !== 'api' ? f.password_login : false,
        password: (f.type !== 'api' && f.password_login && f.password) ? f.password : null,
        user_role_id: Number(f.user_role_id),
        list_role_id: f.list_role_id ? Number(f.list_role_id) : null,
      };
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('usersView', usersView);
  window.Alpine.data('userForm', userForm);
}, { once: true });
