import {
  api,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

// profileView is the logged in user's own profile page.
function profileView() {
  const user = window._user || {};

  return {
    form: {
      name: user.name || '',
      email: user.email || '',
      password: '',
      password2: '',
    },

    // TOTP enable flow.
    showTotpSetup: false,
    totpQR: null,
    totpSecret: null,
    totpCode: '',

    // TOTP disable flow.
    showDisableTotp: false,
    disablePassword: '',

    async onSubmit() {
      const payload = { name: this.form.name, email: this.form.email };

      if (this.form.password) {
        if (this.form.password !== this.form.password2) {
          u.toast(i18n.t('users.passwordMismatch'), 'danger');
          return;
        }
        payload.password = this.form.password;
        payload.password2 = this.form.password2;
      }

      const data = await api('profile.save', '/profile', 'PUT', payload);
      this.form.password = '';
      this.form.password2 = '';
      u.toast(i18n.ts('globals.messages.updated', { name: data.username }));
    },

    // ===============
    // TOTP on/off flow.
    async onEnableTotp() {
      this.showTotpSetup = true;
      try {
        const data = await api('profile.totp', `/users/${user.id}/twofa/totp`, 'GET');
        this.totpQR = data.qr;
        this.totpSecret = data.secret;
        this.$nextTick(() => this.$refs.totpCode && this.$refs.totpCode.focus());
      } catch (e) {
        this.resetTotp();
      }
    },

    async confirmTotp() {
      if (!this.totpCode || this.totpCode.length !== 6) {
        u.toast(i18n.t('globals.messages.invalidValue'), 'danger');
        return;
      }

      await api('profile.totp', `/users/${user.id}/twofa`, 'PUT', {
        secret: this.totpSecret,
        code: this.totpCode,
      });
      u.reload({ message: i18n.t('users.twoFAEnabled') });
    },

    resetTotp() {
      this.showTotpSetup = false;
      this.totpQR = null;
      this.totpSecret = null;
      this.totpCode = '';
    },

    async confirmDisableTotp() {
      if (!this.disablePassword) {
        u.toast(i18n.t('globals.messages.invalidFields'), 'danger');
        return;
      }

      await api('profile.totp', `/users/${user.id}/twofa`, 'DELETE', { password: this.disablePassword });
      u.reload({ message: i18n.t('globals.messages.done') });
    },

    cancelDisableTotp() {
      this.showDisableTotp = false;
      this.disablePassword = '';
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('profileView', profileView);
}, { once: true });
