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
    const base = this.getAttribute('base') || '/admin/static/email-builder';
    const script = new URL(`${base}/email-builder.umd.js`, document.baseURI).href
      .replace(/&/g, '&amp;').replace(/"/g, '&quot;');

    const iframe = document.createElement('iframe');
    iframe.id = 'visual-editor';
    iframe.className = 'visual-editor';
    iframe.title = 'Visual email editor';

    iframe.setAttribute('sandbox', 'allow-scripts allow-downloads allow-forms');
    this.iframe = iframe;
    this._ready = false;
    window.addEventListener('message', this._onMessage, false);
    iframe.srcdoc = `<!DOCTYPE html><html><head><style>
        body { margin: 0; padding: 0; }
        #visual-editor-container { width: 100%; height: 100%; }
      </style></head><body><div id="visual-editor-container"></div>
      <script src="${script}"></script>
      <script>(${visualEditorFrame.toString()})()</script></body></html>`;

    this.appendChild(iframe);
  }

  disconnectedCallback() {
    window.removeEventListener('message', this._onMessage, false);
  }

  render(source) {
    if (!this._ready) {
      this._pending = source;
      return;
    }

    this.iframe.contentWindow.postMessage({ type: 'visualeditor.render', source }, '*');
  }

  insertMedia(url) {
    this.iframe.contentWindow.postMessage({ type: 'visualeditor.insert-media', url }, '*');
  }

  _onMessage(msg) {
    if (!this.iframe || msg.source !== this.iframe.contentWindow) {
      return;
    }
    if (msg.data === 'visualeditor.ready') {
      this._ready = true;
      this.render(this._pending);
      this.dispatchEvent(new CustomEvent('editor-ready', { bubbles: true }));
    } else if (msg.data === 'visualeditor.select-media') {
      this.dispatchEvent(new CustomEvent('visual-media', { bubbles: true }));
    } else if (msg.data?.type === 'visualeditor.change' && typeof msg.data.source === 'string' && typeof msg.data.body === 'string') {
      this.dispatchEvent(new CustomEvent('editor-change', {
        bubbles: true,
        detail: { source: msg.data.source, body: msg.data.body },
      }));
    }
  }
}

function visualEditorFrame() {
  const editor = window.EmailBuilder;
  const container = document.getElementById('visual-editor-container');
  const ready = new MutationObserver(() => {
    ready.disconnect();
    parent.postMessage('visualeditor.ready', '*');
  });
  ready.observe(container, { childList: true });
  editor.render(container.id, {
    data: {},
    onChange: (data, body) => {
      // Fix quotes in Go {{ templating }} that email-builder HTML-escapes.
      body = body.replace(/\{\{[^}]*\}\}/g, (m) => m.replace(/&quot;/g, '"'));
      parent.postMessage({ type: 'visualeditor.change', source: JSON.stringify(data), body }, '*');
    },
  });

  window.addEventListener('message', ({ source, data }) => {
    if (source !== parent || !data) {
      return;
    }
    if (data.type === 'visualeditor.render' && data.source) {
      editor.resetDocument(data.source);
    } else if (data.type === 'visualeditor.insert-media') {
      const input = document.querySelector('.image-url input');
      if (input) {
        const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
        setter.call(input, data.url);
        input.dispatchEvent(new Event('input', { bubbles: true }));
      }
    }
  });
}

if (!customElements.get('visual-editor')) {
  customElements.define('visual-editor', VisualEditor);
}
