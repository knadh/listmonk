import {
  config,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

function escapeAttr(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function component(lists = []) {
  return {
    lists,
    checked: [],
    redirectURL: '',
    html: '',

    init() {
      this.$watch('checked', () => this.renderHTML());
      this.$watch('redirectURL', () => this.renderHTML());
    },

    // ===============
    // Event handlers.
    copyHTML() {
      u.copyToClipboard(this.html);
    },

    // ===============
    // Private functions.
    renderHTML() {
      if (this.checked.length === 0) {
        this.html = '';
        return;
      }

      const sub = config.public_subscription || {};
      const root = config.root_url || '';

      let h = `<form method="post" action="${root}/subscription/form" class="listmonk-form">\n`
        + '  <div>\n'
        + `    <h3>${i18n.t('public.sub')}</h3>\n`
        + '    <input type="hidden" name="nonce" />\n';

      if (this.redirectURL) {
        h += `    <input type="hidden" name="next" value="${escapeAttr(this.redirectURL)}" />\n`;
      }

      h += '\n'
        + `    <p><input type="email" name="email" required placeholder="${i18n.t('subscribers.email')}" /></p>\n`
        + `    <p><input type="text" name="name" placeholder="${i18n.t('public.subName')}" /></p>\n\n`;

      this.checked.forEach((uuid) => {
        const l = this.lists.find((x) => x.uuid === uuid);
        if (!l) {
          return;
        }

        h += '    <p>\n'
          + `      <input id="${l.uuid.substr(0, 5)}" type="checkbox" name="l" checked value="${l.uuid}" />\n`
          + `      <label for="${l.uuid.substr(0, 5)}">${l.name}</label>\n`;

        if (l.description) {
          h += '      <br />\n'
            + `      <span>${l.description}</span>\n`;
        }

        h += '    </p>\n';
      });

      // Captcha?
      if (sub.captcha_enabled) {
        if (sub.captcha_provider === 'altcha') {
          h += '\n'
            + `    <altcha-widget challengeurl="${root}/api/public/captcha/altcha"></altcha-widget>\n`
            + `    <${'script'} type="module" src="${root}/public/static/altcha.umd.js" async defer></${'script'}>\n`;
        } else if (sub.captcha_provider === 'hcaptcha') {
          h += '\n'
            + `    <div class="h-captcha" data-sitekey="${sub.captcha_key}"></div>\n`
            + `    <${'script'} src="https://js.hcaptcha.com/1/api.js" async defer></${'script'}>\n`;
        }
      }

      h += '\n'
        + `    <input type="submit" value="${i18n.t('public.sub')} " />\n`
        + '  </div>\n'
        + '</form>';

      this.html = h;
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('formsView', component);
}, { once: true });
