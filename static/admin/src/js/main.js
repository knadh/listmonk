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
  if (!window.Alpine.store(storeName)) {
    window.Alpine.store(storeName, {
      loading: {},
    });
  }

  // Register reusable components.
  registerTable(window.Alpine, i18n);

  // Register a global Alpine admin app bound to <body> for template helpers.
  window.Alpine.data('adminApp', () => ({
    isLoading,
    copyToClipboard: u.copyToClipboard,
    listAutocomplete,
  }));
}

// Return the global Alpine store.
function getStore() {
  return window.Alpine.store(storeName);
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

    const message = out.message || resp.statusText;
    u.toast(message, 'danger');
    throw new Error(message);
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

  // Init global Alpine component.
  document.addEventListener('alpine:init', setupAlpine, { once: true });
})();
