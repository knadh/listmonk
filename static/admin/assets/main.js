import * as u from './utils.js';

const reParam = /\{([a-z0-9-.]+)\}/ig;

function i18nPath() {
  return (typeof window !== 'undefined' && window._LM_I18NFILE) || '/api/lang/en';
}

export const urls = {
  api: '/api',
  admin: '/admin',
};

export const loading = {};
export const config = {};

export class I18n {
  constructor(path) {
    this.path = path || i18nPath();
    this.lang = {};
  }

  // Public functions.
  async load() {
    if (Object.keys(this.lang).length > 0) {
      return this.lang;
    }

    const resp = await fetch(this.path, { headers: { Accept: 'application/json' } });
    const out = await resp.json();
    Object.assign(this.lang, out && out.data ? out.data : out);
    return this.lang;
  }

  t(key) {
    if (!Object.prototype.hasOwnProperty.call(this.lang, key)) {
      return key;
    }
    return this._singular(this.lang[key]);
  }

  ts(key, params = {}) {
    if (!Object.prototype.hasOwnProperty.call(this.lang, key)) {
      return key;
    }
    return this._tsValue(this._singular(this.lang[key]), params);
  }

  tc(key, n, params = {}) {
    if (!Object.prototype.hasOwnProperty.call(this.lang, key)) {
      return key;
    }

    const out = n > 1 ? this._plural(this.lang[key]) : this._singular(this.lang[key]);
    return this._tsValue(out, { num: n, ...params });
  }

  // Private functions.
  _singular(value) {
    if (!value || !value.includes('|')) {
      return value;
    }
    return value.split('|')[0].trim();
  }

  _plural(value) {
    if (!value || !value.includes('|')) {
      return value;
    }
    const chunks = value.split('|');
    return (chunks[1] || chunks[0]).trim();
  }

  _subAllParams(value) {
    if (!value || !value.includes('{')) {
      return value;
    }
    return value.replace(reParam, (_match, key) => this.t(key));
  }

  _tsValue(value, params = {}) {
    let out = value;
    Object.keys(params).forEach((name) => {
      out = out.replaceAll(`{${name}}`, this._subAllParams(String(params[name])));
    });
    return out;
  }
}

export const i18n = new I18n();
export const lang = i18n.lang;

export async function loadI18n() {
  return i18n.load();
}

export async function loadConfig() {
  if (Object.keys(config).length > 0) {
    return config;
  }

  const data = await api('config', '/config');
  Object.assign(config, data || {});
  return config;
}

export async function init() {
  u.setI18n(i18n);
  await Promise.all([loadConfig(), loadI18n()]);
  return { config, lang };
}

export function api(name, uri, method, data) {
  loading[name] = true;

  return fetch(`${urls.api}${uri}`, {
    method: method || 'GET',
    body: data ? JSON.stringify(data) : null,
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json; charset=utf-8',
    },
  }).then(async (resp) => {
    delete loading[name];

    const out = await resp.json().catch(() => ({}));
    if (resp.ok) {
      return out.data;
    }

    const message = out.message || resp.statusText;
    u.toast(message, 'danger');
    throw new Error(message);
  }).catch((err) => {
    delete loading[name];
    throw err;
  });
}
