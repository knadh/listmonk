import {
  ToastProgrammatic as Toast,
  DialogProgrammatic as Dialog,
} from 'buefy';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const reEmail = /(.+?)@(.+?)/ig;

export default class utils {
  static months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug',
    'Sep', 'Oct', 'Nov', 'Dec'];

  static days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

  // Parses an ISO timestamp to a simpler form.
  static niceDate = (stamp, showTime) => {
    if (!stamp) {
      return '';
    }

    const d = new Date(stamp);
    let out = `${utils.days[d.getDay()]}, ${d.getDate()}`;
    out += ` ${utils.months[d.getMonth()]} ${d.getFullYear()}`;
    if (showTime) {
      out += ` ${d.getHours()}:${d.getMinutes()}`;
    }

    return out;
  };

  static duration(start, end) {
    return dayjs(end).from(dayjs(start), true);
  }

  // Simple, naive, e-mail address check.
  static validateEmail = (e) => e.match(reEmail);

  static niceNumber = (n) => {
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
  static confirm = (msg, onConfirm, onCancel) => {
    Dialog.confirm({
      scroll: 'clip',
      message: !msg ? 'Are you sure?' : msg,
      onConfirm,
      onCancel,
    });
  };

  static prompt = (msg, inputAttrs, onConfirm, onCancel) => {
    Dialog.prompt({
      scroll: 'clip',
      message: msg,
      confirmText: 'OK',
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

  static toast = (msg, typ, duration) => {
    Toast.open({
      message: msg,
      type: !typ ? 'is-success' : typ,
      queue: false,
      duration: duration || 2000,
    });
  };
}
