import {
  ToastProgrammatic as Toast,
  DialogProgrammatic as Dialog,
} from 'buefy';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const reEmail = /(.+?)@(.+?)/ig;

export default class Utils {
  constructor(i18n) {
    this.i18n = i18n;
  }

  // Parses an ISO timestamp to a simpler form.
  niceDate = (stamp, showTime) => {
    if (!stamp) {
      return '';
    }

    const d = new Date(stamp);
    const day = this.i18n.t(`globals.days.${(d.getDay() + 1)}`);
    const month = this.i18n.t(`globals.months.${(d.getMonth() + 1)}`);
    let out = `${day}, ${d.getDate()}`;
    out += ` ${month} ${d.getFullYear()}`;
    if (showTime) {
      out += ` ${d.getHours()}:${d.getMinutes()}`;
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

  // UI shortcuts.
  confirm = (msg, onConfirm, onCancel) => {
    Dialog.confirm({
      scroll: 'clip',
      message: !msg ? this.i18n.t('globals.messages.confirm') : msg,
      confirmText: this.i18n.t('globals.buttons.ok'),
      cancelText: this.i18n.t('globals.buttons.cancel'),
      onConfirm,
      onCancel,
    });
  };

  prompt = (msg, inputAttrs, onConfirm, onCancel) => {
    Dialog.prompt({
      scroll: 'clip',
      message: msg,
      confirmText: this.i18n.t('globals.buttons.ok'),
      cancelText: this.i18n.t('globals.buttons.cancel'),
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
      message: msg,
      type: !typ ? 'is-success' : typ,
      queue: false,
      duration: duration || 2000,
    });
  };
}
