// <visual-editor> is a WebComponent wrapping the self-hosted email-builder UMD
// (loaded on demand into an iframe).
class VisualEditor extends HTMLElement {
  constructor() {
    super();
    this.iframe = null;
    this._ready = false;
    this._pending = undefined;
    this._onMessage = this._onMessage.bind(this);
  }

  connectedCallback() {
    const ta = this.querySelector('textarea');
    const initial = ta ? ta.value : (this.getAttribute('source') || '');
    if (ta) {
      ta.hidden = true;
    }

    if (initial) {
      try { this._pending = JSON.parse(initial); } catch (e) { this._pending = null; }
    }
    this._base = this.getAttribute('base') || '/admin/static/email-builder';

    const iframe = document.createElement('iframe');
    iframe.id = 'visual-editor';
    iframe.className = 'visual-editor';
    iframe.title = 'Visual email editor';
    this.appendChild(iframe);
    this.iframe = iframe;

    iframe.srcdoc = `<!DOCTYPE html><html><head><style>
        body { margin: 0; padding: 0; }
        #visual-editor-container { width: 100%; height: 100%; }
      </style></head><body><div id="visual-editor-container"></div></body></html>`;

    iframe.onload = () => {
      this._loadScript().then(() => {
        // Apply the latest requested source (which may have been updated by the
        // host after a content-type conversion while the iframe was loading).
        this._ready = true;
        this.render(this._pending);
        // Signal the host that the email-builder is loaded and can be revealed.
        this.dispatchEvent(new CustomEvent('editor-ready', { bubbles: true }));
      }).catch((err) => {
        // eslint-disable-next-line no-console
        console.error('Failed to load email-builder script:', err);
      });
    };

    window.addEventListener('message', this._onMessage, false);
  }

  disconnectedCallback() {
    window.removeEventListener('message', this._onMessage, false);
  }

  _loadScript() {
    return new Promise((resolve, reject) => {
      const win = this.iframe.contentWindow;
      if (win.EmailBuilder) {
        resolve();
        return;
      }
      const s = this.iframe.contentDocument.createElement('script');
      s.id = 'email-builder-script';
      s.src = `${this._base}/email-builder.umd.js`;
      s.onload = () => resolve();
      s.onerror = reject;
      this.iframe.contentDocument.head.appendChild(s);
    });
  }

  render(source) {
    if (!this._ready) {
      this._pending = source;
      return;
    }

    const win = this.iframe.contentWindow;
    const em = win.EmailBuilder;
    if (!em) {
      return;
    }

    if (!em.isRendered('visual-editor-container')) {
      em.render('visual-editor-container', {
        data: {},
        onChange: (data, body) => {
          // Fix quotes in Go {{ templating }} that email-builder HTML-escapes.
          const tpl = body.replace(/\{\{[^}]*\}\}/g, (m) => m.replace(/&quot;/g, '"'));
          this.dispatchEvent(new CustomEvent('editor-change', {
            bubbles: true,
            detail: { source: JSON.stringify(data), body: tpl },
          }));
        },
      });
    }

    if (!source) {
      return;
    }

    // Brute-force wait for the container to render, then reset the document.
    let n = 0;
    const timer = window.setInterval(() => {
      const container = this.iframe.contentWindow.document.getElementById('visual-editor-container');
      if (container && container.hasChildNodes()) {
        em.resetDocument(source);
        window.clearInterval(timer);
        return;
      }
      n += 1;
      if (n > 10) {
        window.clearInterval(timer);
      }
    }, 100);
  }

  // Inject a media URL into the email-builder sidebar's image URL input.
  insertMedia(url) {
    const input = this.iframe.contentDocument.querySelector('.image-url input');
    if (!input) {
      return;
    }
    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
    setter.call(input, url);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  }

  _onMessage(msg) {
    if (msg.data === 'visualeditor.select-media') {
      this.dispatchEvent(new CustomEvent('visual-media', { bubbles: true }));
    }
  }
}

if (!customElements.get('visual-editor')) {
  customElements.define('visual-editor', VisualEditor);
}
