export class I18n {
  constructor(lang = {}) {
    this.lang = { ...lang };
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
