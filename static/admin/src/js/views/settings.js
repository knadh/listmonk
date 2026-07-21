import {
  api,
  config,
  i18n,
  urls,
} from '../main.js';
import * as u from '../utils.js';

// Quick-fill templates for known SMTP providers.
const smtpTemplates = {
  gmail: { host: 'smtp.gmail.com', port: 465, auth_protocol: 'login', tls_type: 'TLS' },
  ses: { host: 'email-smtp.YOUR-REGION.amazonaws.com', port: 465, auth_protocol: 'login', tls_type: 'TLS' },
  azure: { host: 'smtp.azurecomm.net', port: 587, auth_protocol: 'login', tls_type: 'STARTTLS' },
  mailjet: { host: 'in-v3.mailjet.com', port: 465, auth_protocol: 'cram', tls_type: 'TLS' },
  mailgun: { host: 'smtp.mailgun.org', port: 465, auth_protocol: 'login', tls_type: 'TLS' },
  sendgrid: { host: 'smtp.sendgrid.net', port: 465, auth_protocol: 'login', tls_type: 'TLS' },
  forwardemail: { host: 'smtp.forwardemail.net', port: 465, auth_protocol: 'login', tls_type: 'TLS' },
  postmark: { host: 'smtp.postmarkapp.com', port: 587, auth_protocol: 'cram', tls_type: 'STARTTLS' },
  lettermint: { host: 'smtp.lettermint.co', port: 465, auth_protocol: 'login', tls_type: 'TLS' },
};

const OIDC_PROVIDERS = {
  google: 'https://accounts.google.com',
  microsoft: 'https://login.microsoftonline.com/{TENANT_HERE}/v2.0',
  apple: 'https://appleid.apple.com',
};

function isDummy(pwd) {
  return !pwd || (pwd.match(/•/g) || []).length === pwd.length;
}

function hasDummy(pwd) {
  return pwd.includes('•');
}

// Persist the open/closed state of SMTP/messenger accordions.
const OPEN_KEY = 'settings.openBoxes';
function loadOpen() {
  try {
    return JSON.parse(localStorage.getItem(OPEN_KEY)) || {};
  } catch (e) {
    return {};
  }
}

function component(settings, userRoles, listRoles) {
  const openState = loadOpen();

  // Normalize the raw settings into a display-friendly form (mirrors the old getSettings()).
  const normalize = (data) => {
    const d = JSON.parse(JSON.stringify(data));

    // Serialize the email_headers map to a display string.
    for (let i = 0; i < d.smtp.length; i += 1) {
      d.smtp[i].strEmailHeaders = JSON.stringify(d.smtp[i].email_headers || [], null, 4);
      d.smtp[i].showHeaders = false;
      d.smtp[i].open = !!openState.smtp?.[i];
    }

    // Messengers render as collapsible accordions too.
    (d.messengers || []).forEach((m, i) => { m.open = !!openState.messengers?.[i]; });

    // Domain block/allow lists, convert array to multi-line string.
    d['privacy.domain_blocklist'] = (d['privacy.domain_blocklist'] || []).join('\n');
    d['privacy.domain_allowlist'] = (d['privacy.domain_allowlist'] || []).join('\n');

    // Ensure there's always at least one bounce mailbox to bind the enable switch to.
    if (!Array.isArray(d['bounce.mailboxes']) || d['bounce.mailboxes'].length === 0) {
      d['bounce.mailboxes'] = [{
        enabled: false, type: 'pop', host: '', port: 995, auth_protocol: 'userpass',
        username: '', password: '', return_path: '', tls_enabled: true, tls_skip_verify: false,
        scan_interval: '15m',
      }];
    }

    return d;
  };

  const form = normalize(settings);

  // Keep search index+DOM references outside of Alpine.
  let searchIndex = [];
  let tabButtons = [];
  let tabPanels = [];

  return {
    form,
    userRoles,
    listRoles,

    smtpTestItem: null,
    testEmail: '',
    errMsg: '',

    init() {
      this.$nextTick(() => this.buildSearchIndex());
    },

    // ===============
    // Search.
    // Index every keyword tagged block and its tab.
    buildSearchIndex() {
      const tabs = this.$refs.tabs;
      if (!tabs) {
        return;
      }

      tabButtons = Array.from(tabs.querySelectorAll(':scope > [role="tablist"] > [role="tab"]'));
      tabPanels = Array.from(tabs.querySelectorAll(':scope > [role="tabpanel"]'));
      searchIndex = [];
      tabPanels.forEach((p, i) => {
        p.querySelectorAll('[data-keywords]').forEach((el) => {
          searchIndex.push({ el, panelIndex: i, keywords: (el.dataset.keywords || '').toLowerCase() });
        });
      });
    },

    onSearch(e) {
      const q = (e.target.value || '').trim().toLowerCase();
      const terms = q.split(/\s+/).filter(Boolean);

      // If there are no matches, then all settings are shown as nromal.
      const panels = new Set();
      const matches = searchIndex.map((b) => {
        const ok = terms.length > 0 && terms.every((t) => b.keywords.includes(t));
        if (ok) {
          panels.add(b.panelIndex);
        }
        return ok;
      });

      if (panels.size === 0) {
        this.resetSearch();
        return;
      }

      this.$root.classList.add('is-searching');
      searchIndex.forEach((b, i) => b.el.classList.toggle('search-hide', !matches[i]));

      // Show only panels+tabs that have at least one matching block.
      tabPanels.forEach((p, i) => p.classList.toggle('search-match', panels.has(i)));
      tabButtons.forEach((t, i) => t.classList.toggle('search-hide', !panels.has(i)));
    },

    // Undo all search filters and render tabs and panels as normal.
    resetSearch() {
      this.$root.classList.remove('is-searching');
      searchIndex.forEach((b) => b.el.classList.remove('search-hide'));
      tabButtons.forEach((t) => t.classList.remove('search-hide'));
      tabPanels.forEach((p) => p.classList.remove('search-match'));
    },

    clearSearch() {
      this.$refs.search.value = '';
      this.resetSearch();
    },

    // Clicking a tab while searching resets the search and navigates to that tab.
    onTabClick(e) {
      if (e.target.closest('[role="tab"]') && this.$root.classList.contains('is-searching')) {
        this.clearSearch();
      }
    },

    // ot-tabs handles left/right arrows. Since this is vertical, handle up/down.
    onTabKeydown(e) {
      const dir = { ArrowUp: -1, ArrowDown: 1 }[e.key];
      if (!dir || !e.target.closest('[role="tab"]')) {
        return;
      }
      e.preventDefault();

      const tabs = this.$refs.tabs;
      const btns = tabs.querySelectorAll(':scope > [role="tablist"] > [role="tab"]');
      const next = (tabs.activeIndex + dir + btns.length) % btns.length;

      tabs.activeIndex = next;
      btns[next].focus();
    },

    // ===============
    // Getters / setters.
    get isURLOk() {
      try {
        const url = new URL(config.root_url);
        return url.hostname !== 'localhost' && url.hostname !== '127.0.0.1';
      } catch (e) {
        return false;
      }
    },

    get captchaEnabled() {
      return this.form['security.captcha'].altcha.enabled || this.form['security.captcha'].hcaptcha.enabled;
    },
    set captchaEnabled(v) {
      this.form['security.captcha'].altcha.enabled = !!v;
      this.form['security.captcha'].hcaptcha.enabled = false;
    },

    get captchaProvider() {
      return this.form['security.captcha'].hcaptcha.enabled ? 'hcaptcha' : 'altcha';
    },
    set captchaProvider(v) {
      this.form['security.captcha'].hcaptcha.enabled = v === 'hcaptcha';
      this.form['security.captcha'].altcha.enabled = v === 'altcha';
    },

    get trustedURLs() {
      const d = this.form['security.trusted_urls'];
      return Array.isArray(d) ? d.join('\n') : '';
    },
    set trustedURLs(v) {
      this.form['security.trusted_urls'] = v.split('\n');
    },

    // ===============
    // Event handlers.
    countLines(str) {
      return (str || '').split('\n').filter((l) => l.trim()).length;
    },

    // Toggle an andpersist accordion box open/closed state.
    setOpen(group, n, open) {
      this.form[group][n].open = open;
      (openState[group] ??= {})[n] = open;
      localStorage.setItem(OPEN_KEY, JSON.stringify(openState));
    },

    setOIDCProvider(provider) {
      this.form['security.oidc'].provider_url = OIDC_PROVIDERS[provider];
      this.form['security.oidc'].provider_name = provider.charAt(0).toUpperCase() + provider.slice(1);
      this.$nextTick(() => this.$refs.oidcClientID?.focus());
    },

    onS3URLChange() {
      // Don't touch a custom non-AWS URL.
      if (this.form['upload.s3.url'] !== '' && !this.form['upload.s3.url'].match(/amazonaws\.com/)) {
        return;
      }
      this.form['upload.s3.url'] = `https://s3.${this.form['upload.s3.aws_default_region']}.amazonaws.com`;
    },

    // ===============
    // SMTP.
    addSMTP() {
      this.form.smtp.push({
        name: '', enabled: true, host: '', hello_hostname: '', port: 587, auth_protocol: 'none',
        username: '', password: '', email_headers: [], strEmailHeaders: '[]', showHeaders: false,
        from_addresses: [], max_conns: 10, max_msg_retries: 2, msg_retry_delay: '10ms',
        idle_timeout: '15s', wait_timeout: '5s', tls_type: 'STARTTLS', tls_skip_verify: false,
        open: true,
      });
      this.$nextTick(() => {
        const items = document.querySelectorAll('.mail-servers input[placeholder="smtp.yourmailserver.net"]');
        items[items.length - 1]?.focus();
      });
    },

    async removeSMTP(n) {
      if (!(await u.confirm())) {
        return;
      }
      this.form.smtp.splice(n, 1);
    },

    fillSMTP(n, key) {
      this.form.smtp[n] = {
        ...this.form.smtp[n],
        ...smtpTemplates[key],
        username: '',
        password: '',
        hello_hostname: '',
        tls_skip_verify: false,
      };
      this.$nextTick(() => document.querySelector(`.smtp-username-${n}`)?.focus());
    },

    showSMTPTest(n) {
      this.smtpTestItem = this.smtpTestItem === n ? null : n;
      this.errMsg = '';
      if (this.smtpTestItem === n) {
        // The test box lives inside the accordion body, so make sure it's open.
        this.form.smtp[n].open = true;
        this.$nextTick(() => document.querySelector(`.test-email-${n}`)?.focus());
      }
    },

    isTestEnabled(item) {
      if (!item.host || !item.port) {
        return false;
      }
      if (item.auth_protocol !== 'none' && hasDummy(item.password)) {
        return false;
      }
      return true;
    },

    async doSMTPTest(item, n) {
      if (!this.isTestEnabled(item)) {
        u.toast(i18n.t('settings.smtp.testEnterEmail'), 'warning');
        this.$nextTick(() => {
          const el = document.querySelector(`.smtp-password-${n}`);
          this.form.smtp[n].password = '';
          el?.focus();
        });
        return;
      }

      this.errMsg = '';
      try {
        await api('settings.testSMTP', '/settings/smtp/test', 'POST', { ...item, email: this.testEmail });
        u.toast(i18n.t('campaigns.testSent'));
      } catch (err) {
        this.errMsg = err.message;
      }
    },

    // ===============
    // Messengers.
    addMessenger() {
      this.form.messengers.push({
        enabled: true, root_url: '', name: '', username: '', password: '',
        max_conns: 25, max_msg_retries: 2, timeout: '5s', open: true,
      });
      this.$nextTick(() => {
        const items = document.querySelectorAll('.messengers input[placeholder="mymessenger"]');
        items[items.length - 1]?.focus();
      });
    },

    async removeMessenger(n) {
      if (!(await u.confirm())) {
        return;
      }
      this.form.messengers.splice(n, 1);
    },

    // ===============
    // Submit.
    async onSubmit() {
      const form = JSON.parse(JSON.stringify(this.form));
      let dummy = '';

      // SMTP servers.
      for (let i = 0; i < form.smtp.length; i += 1) {
        form.smtp[i].host = form.smtp[i].host?.trim();
        if (isDummy(form.smtp[i].password)) {
          form.smtp[i].password = '';
        } else if (hasDummy(form.smtp[i].password)) {
          dummy = `smtp #${i + 1}`;
        }
        form.smtp[i].email_headers = (form.smtp[i].strEmailHeaders && form.smtp[i].strEmailHeaders !== '[]')
          ? JSON.parse(form.smtp[i].strEmailHeaders) : [];
      }

      // Bounce mailboxes.
      for (let i = 0; i < form['bounce.mailboxes'].length; i += 1) {
        form['bounce.mailboxes'][i].host = form['bounce.mailboxes'][i].host?.trim();
        if (isDummy(form['bounce.mailboxes'][i].password)) {
          form['bounce.mailboxes'][i].password = '';
        } else if (hasDummy(form['bounce.mailboxes'][i].password)) {
          dummy = `bounce #${i + 1}`;
        }
      }

      // Standalone secrets.
      const secrets = [
        ['upload.s3.aws_secret_access_key', form, 's3'],
        ['bounce.sendgrid_key', form, 'sendgrid'],
      ];
      secrets.forEach(([key, obj, name]) => {
        if (isDummy(obj[key])) {
          obj[key] = '';
        } else if (hasDummy(obj[key])) {
          dummy = name;
        }
      });

      const nested = [
        [form['bounce.azure'], 'shared_secret', 'azure shared secret'],
        [form['security.captcha'].hcaptcha, 'secret', 'captcha'],
        [form['security.oidc'], 'client_secret', 'oidc'],
        [form['bounce.postmark'], 'password', 'postmark'],
        [form['bounce.forwardemail'], 'key', 'forwardemail'],
        [form['bounce.lettermint'], 'key', 'lettermint'],
      ];
      nested.forEach(([obj, key, name]) => {
        if (isDummy(obj[key])) {
          obj[key] = '';
        } else if (hasDummy(obj[key])) {
          dummy = name;
        }
      });

      for (let i = 0; i < form.messengers.length; i += 1) {
        if (isDummy(form.messengers[i].password)) {
          form.messengers[i].password = '';
        } else if (hasDummy(form.messengers[i].password)) {
          dummy = `messenger #${i + 1}`;
        }
      }

      if (dummy) {
        u.toast(i18n.ts('globals.messages.passwordChangeFull', { name: dummy }), 'warning');
        return;
      }

      // Convert domain lists multi-line string back to arrays.
      form['privacy.domain_blocklist'] = form['privacy.domain_blocklist'].split('\n').map((v) => v.trim().toLowerCase()).filter((v) => v !== '');
      form['privacy.domain_allowlist'] = form['privacy.domain_allowlist'].split('\n').map((v) => v.trim().toLowerCase()).filter((v) => v !== '');

      // OIDC role ids.
      form['security.oidc'].default_user_role_id = form['security.oidc'].default_user_role_id
        ? parseInt(form['security.oidc'].default_user_role_id, 10) : null;
      form['security.oidc'].default_list_role_id = form['security.oidc'].default_list_role_id
        ? parseInt(form['security.oidc'].default_list_role_id, 10) : null;

      // Strip display-only fields.
      form.smtp.forEach((s) => { delete s.strEmailHeaders; delete s.showHeaders; });

      await api('settings', '/settings', 'PUT', form);
      await this._awaitRestart();
    },

    // Poll the health endpoint until the app is back up after a settings-triggered
    // restart, then reload the page.
    _awaitRestart() {
      u.toast(i18n.ts('globals.messages.updated', { name: i18n.t('settings.title') }));
      return new Promise((resolve) => {
        const poll = setInterval(() => {
          fetch(`${urls.api}/health`).then((r) => {
            if (r.ok) {
              clearInterval(poll);
              resolve();
            }
          }).catch(() => { });
        }, 500);
      });
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('settingsView', component);
}, { once: true });
