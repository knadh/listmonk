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

const prefKey = 'listmonk_pref';

let i18n = {
  t: (key) => key,
};

export function setI18n(instance) {
  i18n = instance || i18n;
}

function _toastVariant(typ) {
  if (!typ) {
    return 'success';
  }

  return {
    'is-success': 'success',
    success: 'success',
    'is-danger': 'danger',
    danger: 'danger',
    error: 'danger',
    'is-warning': 'warning',
    warning: 'warning',
  }[typ] || 'info';
}

export function toast(message, variant, duration) {
  if (window.ot && window.ot.toast) {
    window.ot.toast(message, '', {
      duration: duration || 3000,
      placement: 'top-center',
      variant: _toastVariant(variant),
    });
    return;
  }
  window.alert(message);
}

export function confirm(message) {
  const tpl = document.getElementById('confirm-dialog-template');
  const fallback = message || i18n.t('globals.messages.confirm');
  if (!tpl) {
    return Promise.resolve(window.confirm(fallback));
  }

  return new Promise((resolve) => {
    const dialog = tpl.content.firstElementChild.cloneNode(true);
    const msg = dialog.querySelector('[data-confirm-message]');
    const btnCancel = dialog.querySelector('[data-confirm-cancel]');
    const btnOk = dialog.querySelector('[data-confirm-ok]');

    msg.textContent = fallback;
    btnCancel.textContent = i18n.t('globals.buttons.cancel');
    btnOk.textContent = i18n.t('globals.buttons.ok');

    let resolved = false;
    const done = (ok) => {
      if (resolved) {
        return;
      }
      resolved = true;
      resolve(ok);
      dialog.close();
      dialog.remove();
    };

    btnCancel.addEventListener('click', () => done(false));
    btnOk.addEventListener('click', () => done(true));
    dialog.addEventListener('cancel', (e) => {
      e.preventDefault();
      done(false);
    });
    dialog.addEventListener('close', () => done(false));

    document.body.appendChild(dialog);
    dialog.showModal();
    btnOk.focus();
  });
}

export function checkedValues(selector) {
  return Array.from(document.querySelectorAll(`${selector}:checked`)).map((el) => Number(el.value));
}

export function validateEmail(email) {
  return /(.+?)@(.+?)/ig.test(email);
}

export function niceNumber(n) {
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

  const out = n / div;
  return Number.isInteger(out) ? out + pfx : out.toFixed(2) + pfx;
}

export function formatNumber(value) {
  return new Intl.NumberFormat().format(value);
}

export function parseQueryIDs(ids) {
  if (!ids) {
    return [];
  }
  if (typeof ids === 'string' || typeof ids === 'number') {
    return [parseInt(ids, 10)];
  }
  return ids.map((id) => parseInt(id, 10));
}

export function escapeHTML(html) {
  return html.replace(/[&<>"'`=/]/g, (s) => htmlEntities[s]);
}

export function titleCase(str) {
  return str[0].toUpperCase() + str.substr(1).toLowerCase();
}

export function tdID(row) {
  return { 'data-id': row.id.toString() };
}

export function getPref(key) {
  if (localStorage.getItem(prefKey) === null) {
    return null;
  }
  const prefs = JSON.parse(localStorage.getItem(prefKey));
  return key in prefs ? prefs[key] : null;
}

export function setPref(key, value) {
  let prefs = {};
  if (localStorage.getItem(prefKey) !== null) {
    prefs = JSON.parse(localStorage.getItem(prefKey));
  }
  prefs[key] = value;
  localStorage.setItem(prefKey, JSON.stringify(prefs));
}

// Reload the page. If a toast object is given, it's put in
// sessionStorage and shown as a toast after the reload.
export function reload(toast) {
  if (toast) {
    sessionStorage.setItem('reload-toast', JSON.stringify(toast));
  }

  window.location.reload();
}


export function copyToClipboard(text) {
  if (!navigator.clipboard) {
    return;
  }

  navigator.clipboard.writeText(text).then(() => {
    toast(i18n.t('globals.messages.copied'), 'success');
  });
};