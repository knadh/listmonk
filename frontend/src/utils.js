import {
  ToastProgrammatic as Toast,
  DialogProgrammatic as Dialog,
} from 'buefy';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import updateLocale from 'dayjs/plugin/updateLocale';

dayjs.extend(updateLocale);
dayjs.extend(relativeTime);

const reEmail = /(.+?)@(.+?)/ig;
const prefKey = 'listmonk_pref';

const htmlEntities = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
  '/': '&#x2F;',
  '`': '&#x60;',
  '=': '&#x3D;',
};

export default class Utils {
  constructor(i18n) {
    this.i18n = i18n;
    this.intlNumFormat = new Intl.NumberFormat();

    if (i18n) {
      dayjs.updateLocale('en', {
        relativeTime: {
          future: '%s',
          past: '%s',
          s: `${i18n.tc('globals.terms.second', 2)}`,
          m: `1 ${i18n.tc('globals.terms.minute', 1)}`,
          mm: `%d ${i18n.tc('globals.terms.minute', 2)}`,
          h: `1 ${i18n.tc('globals.terms.hour', 1)}`,
          hh: `%d ${i18n.tc('globals.terms.hour', 2)}`,
          d: `1 ${i18n.tc('globals.terms.day', 1)}`,
          dd: `%d ${i18n.tc('globals.terms.day', 2)}`,
          M: `1 ${i18n.tc('globals.terms.month', 1)}`,
          MM: `%d ${i18n.tc('globals.terms.month', 2)}`,
          y: `${i18n.tc('globals.terms.year', 1)}`,
          yy: `%d ${i18n.tc('globals.terms.year', 2)}`,
        },
      });
    }
  }

  getDate = (d) => dayjs(d);

  // Parses an ISO timestamp to a simpler form.
  niceDate = (stamp, showTime) => {
    if (!stamp) {
      return '';
    }

    const d = dayjs(stamp);
    const day = this.i18n.t(`globals.days.${d.day() + 1}`);
    const month = this.i18n.t(`globals.months.${d.month() + 1}`);
    let out = d.format(`[${day},] DD [${month}] YYYY`);
    if (showTime) {
      out += d.format(', HH:mm');
    }

    return out;
  };

  duration = (start, end) => dayjs(end).from(dayjs(start), true);

  // Simple, naive, e-mail address check.
  validateEmail = (e) => e.match(reEmail);

  niceNumber = (n) => {
    if (n === null || n === undefined) {
      return 0;
    }

    let pfx = '';
    let div = 1;

    if (n >= 1.0e+9) {
      pfx = 'b';
      div = 1.0e+9;
    } else if (n >= 1.0e+6) {
      pfx = 'm';
      div = 1.0e+6;
    } else if (n >= 1.0e+4) {
      pfx = 'k';
      div = 1.0e+3;
    } else {
      return n;
    }

    // Whole number without decimals.
    const out = (n / div);
    if (Math.floor(out) === n) {
      return out + pfx;
    }

    return out.toFixed(2) + pfx;
  }

  formatNumber(v) {
    return this.intlNumFormat.format(v);
  }

  // Parse one or more numeric ids as query params and return as an array of ints.
  parseQueryIDs = (ids) => {
    if (!ids) {
      return [];
    }

    if (typeof ids === 'string') {
      return [parseInt(ids, 10)];
    }

    if (typeof ids === 'number') {
      return [parseInt(ids, 10)];
    }

    return ids.map((id) => parseInt(id, 10));
  }

  // https://stackoverflow.com/a/12034334
  escapeHTML = (html) => html.replace(/[&<>"'`=/]/g, (s) => htmlEntities[s]);

  titleCase = (str) => str[0].toUpperCase() + str.substr(1).toLowerCase();

  // UI shortcuts.
  confirm = (msg, onConfirm, onCancel) => {
    Dialog.confirm({
      scroll: 'keep',
      message: !msg ? this.i18n.t('globals.messages.confirm') : this.escapeHTML(msg),
      confirmText: this.i18n.t('globals.buttons.ok'),
      cancelText: this.i18n.t('globals.buttons.cancel'),
      onConfirm,
      onCancel,
    });
  };

  prompt = (msg, inputAttrs, onConfirm, onCancel, params) => {
    const p = params || {};

    Dialog.prompt({
      scroll: 'keep',
      message: this.escapeHTML(msg),
      confirmText: p.confirmText || this.i18n.t('globals.buttons.ok'),
      cancelText: p.cancelText || this.i18n.t('globals.buttons.cancel'),
      inputAttrs: {
        type: 'string',
        maxlength: 200,
        ...inputAttrs,
      },
      trapFocus: true,
      onConfirm,
      onCancel,
    });
  };

  toast = (msg, typ, duration) => {
    Toast.open({
      message: this.escapeHTML(msg),
      type: !typ ? 'is-success' : typ,
      queue: false,
      duration: duration || 3000,
      position: 'is-top',
      pauseOnHover: true,
    });
  };

  // Takes a props.row from a Buefy b-column <td> template and
  // returns a `data-id` attribute which Buefy then applies to the td.
  tdID = (row) => ({ 'data-id': row.id.toString() });

  camelString = (str) => {
    const s = str.replace(/[-_\s]+(.)?/g, (match, chr) => (chr ? chr.toUpperCase() : ''));
    return s.substr(0, 1).toLowerCase() + s.substr(1);
  }

  // camelKeys recursively camelCases all keys in a given object (array or {}).
  // For each key it traverses, it passes a dot separated key path to an optional testFunc() bool.
  // so that it can camelcase or leave a particular key alone based on what testFunc() returns.
  // eg: The keypath for {"data": {"results": ["created_at": 123]}} is
  // .data.results.*.created_at (array indices become *)
  // testFunc() can examine this key and return true to convert it to camelcase
  // or false to leave it as-is.
  camelKeys = (obj, testFunc, keys) => {
    if (obj === null) {
      return obj;
    }

    if (Array.isArray(obj)) {
      return obj.map((o) => this.camelKeys(o, testFunc, `${keys || ''}.*`));
    }

    if (obj.constructor === Object) {
      return Object.keys(obj).reduce((result, key) => {
        const keyPath = `${keys || ''}.${key}`;
        let k = key;

        // If there's no testfunc or if a function is defined and it returns true, convert.
        if (testFunc === undefined || testFunc(keyPath)) {
          k = this.camelString(key);
        }

        return {
          ...result,
          [k]: this.camelKeys(obj[key], testFunc, keyPath),
        };
      }, {});
    }

    return obj;
  };

  getPref = (key) => {
    if (localStorage.getItem(prefKey) === null) {
      return null;
    }

    const p = JSON.parse(localStorage.getItem(prefKey));
    return key in p ? p[key] : null;
  };

  setPref = (key, val) => {
    let p = {};
    if (localStorage.getItem(prefKey) !== null) {
      p = JSON.parse(localStorage.getItem(prefKey));
    }

    p[key] = val;
    localStorage.setItem(prefKey, JSON.stringify(p));
  }
}
