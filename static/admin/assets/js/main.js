import Alpine from 'alpinejs';
import '@knadh/oat';
import { I18n } from './i18n.js';
import { register as registerTable } from './table.js';
import * as u from './utils.js';

// ==============================
// Private constants.
const reParam = /\{([a-z0-9-.]+)\}/ig;
const storeName = 'admin';

// ========
// Private functions.
function setupAlpine() {
  // Register global store.
  if (!Alpine.store(storeName)) {
    Alpine.store(storeName, {
      loading: {},
    });
  }

  // Register reusable components.
  registerTable(i18n);

  // Register a global Alpine admin app bound to <body> for template helpers.
  Alpine.data('adminApp', () => ({
    isLoading,
    copyToClipboard: u.copyToClipboard,
    listAutocomplete,

    theme: localStorage.getItem('theme') || 'auto',

    setTheme(mode) {
      this.theme = mode;

      if (mode === 'auto') {
        localStorage.removeItem('theme');
        document.body.style.colorScheme = '';
      } else {
        localStorage.setItem('theme', mode);
        document.body.style.colorScheme = mode;
      }
    },

    // Trigger an app restart (from the "needs restart" notice), then poll the
    // health endpoint until the app is back up and reload the page.
    async restartApp() {
      if (!(await u.confirm(i18n.t('settings.confirmRestart')))) {
        return;
      }

      try {
        await api('reload', '/admin/reload', 'POST');
      } catch {
        return;
      }

      u.toast(i18n.ts('globals.messages.updated', { name: i18n.t('settings.title') }));
      const poll = setInterval(() => {
        fetch(`${urls.api}/health`).then((r) => {
          if (r.ok) {
            clearInterval(poll);
            window.location.reload();
          }
        }).catch(() => { });
      }, 500);
    },
  }));
}

// Subscribe to the server's error event stream and toast server-side errors as
// they happen. The stream is only advertised (data-error-events on <body>) to
// users who have permission to read it.
function listenErrorEvents() {
  if (!document.body.hasAttribute('data-error-events')) {
    return;
  }

  // Extract the message out of `file.go:123: message` log lines.
  const reMatchLog = /(.+?)\.go:\d+:(.+?)$/im;
  const src = new EventSource(`${urls.api}/events?type=error`, { withCredentials: true });

  let numEv = 0;
  src.onmessage = (e) => {
    // Cap the number of toasts to not flood the UI on a burst of errors.
    if (numEv >= 50) {
      src.close();
      return;
    }
    numEv += 1;

    let d = null;
    try {
      d = JSON.parse(e.data);
    } catch {
      return;
    }

    if (!d || d.type !== 'error' || !d.message) {
      return;
    }

    const msg = reMatchLog.exec(d.message.trim());
    u.toast(msg ? msg[2].trim() : d.message, 'error');
  };
}

// Return the global Alpine store.
function getStore() {
  return Alpine.store(storeName);
}


// ==============================
// Public constants.
export const urls = {
  api: '/api',
  admin: '/admin',
};

// Server config.
export const config = {
  ...((typeof window !== 'undefined' && window._LM_CONFIG) || {}),
};

// Initialize i18n and attach it to utils.
export const i18n = new I18n((typeof window !== 'undefined' && window._LM_I18N) || {});
u.setI18n(i18n);


// ========
// Public functions.

// ListTag is the object used in the <ot-taginput> list selector.
export class ListTag {
  constructor(list) {
    Object.assign(this, list);
  }

  toString() {
    return this.name;
  }
}

// listAutocomplete populates the <ot-taginput> list selector's suggestions with lists.
export function listAutocomplete(el) {
  const ti = el.closest('ot-taginput');
  const chosen = new Set((ti ? ti.value : []).map((l) => l.id));
  const q = el.value.toLowerCase();

  el.list.replaceChildren(...(window._lists || [])
    .filter((l) => !chosen.has(l.id) && l.name.toLowerCase().includes(q))
    // Cap suggestions (so focusing with an empty input still shows a usable list of items).
    .slice(0, 10)
    .map((l) => {
      const o = new Option(l.name);
      o.data = new ListTag(l);
      return o;
    }));
}

// Returns true if the given named api() call is currently loading.
export function isLoading(name) {
  return getStore().loading[name] === true;
}

// JSON API-request helper. Shows a toast on errors and also sets the loading{} states
// for the given request name which can be used for loading spinners on the UI.
export async function api(name, uri, method, data) {
  getStore().loading[name] = true;

  return fetch(`${urls.api}${uri}`, {
    method: method || 'GET',
    body: data ? JSON.stringify(data) : null,
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json; charset=utf-8',
    },
  }).then(async (resp) => {
    getStore().loading[name] = false;

    const out = await resp.json().catch(() => ({}));
    if (resp.ok) {
      return out.data;
    }

    // Throw with the server message; the single .catch below shows the toast.
    throw new Error(out.message || resp.statusText);
  }).catch((err) => {
    getStore().loading[name] = false;
    u.toast(err.message, 'danger');
    throw err;
  });
}

// ========
(() => {
  // Show a toast if there's a reload-toast in sessionStorage.
  const toast = sessionStorage.getItem('reload-toast');
  if (toast) {
    try {
      u.toast(...Object.values(JSON.parse(toast)));
    } catch (err) { }
    sessionStorage.removeItem('reload-toast');
  }

  // Initialize Alpine after everything else is loaded.
  window.Alpine = Alpine;
  document.addEventListener('alpine:init', setupAlpine, { once: true });
  if (document.readyState === 'complete') {
    Alpine.start();
  } else {
    document.addEventListener('DOMContentLoaded', () => Alpine.start(), { once: true });
  }

  listenErrorEvents();
})();
