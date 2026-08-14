// <richtext-editor> is a WebComponent wrapping a self-hosted TinyMCE 5 instance,
// lazy-loaded on demand (like code-editor). It reads its initial value from a
// nested <textarea>, exposes a `value` property, and emits `input` on change.
//
// Bespoke behaviours ported from the old Vue RichtextEditor:
//   - "Source code" and "Insert HTML" toolbar buttons open a native <dialog>
//     with a CodeMirror <code-editor> and a "Format HTML" (beautify) button.
//   - "Track link?" checkbox in the link dialog (appends @TrackLink).
//   - "Embed" checkbox in the image dialog (sets data-embed).
//   - Media image picker via a bubbling `richtext-media` event (host supplies a URL).
//   - Ctrl+S / F9 bubble as `campaign-save` / `campaign-preview` events.
import { beautifyHTML } from './content.js';

const CONFIG = (typeof window !== 'undefined' && window._LM_CONFIG) || {};

// Map of listmonk language codes to TinyMCE language files (shipped by tinymce-i18n,
// copied into dist/tinymce/lang/ at build time — keep in sync with TINY_LANGS in build.mjs).
// es -> es_MX: tinymce-i18n no longer ships the old es_419 (Latin American) pack.
const LANGS = {
  cs: 'cs', de: 'de', es: 'es_MX', fr: 'fr_FR', it: 'it_IT',
  pl: 'pl', pt: 'pt_PT', 'pt-BR': 'pt_BR', ro: 'ro', tr: 'tr',
};

const TRACK_LINK = 'trackLink';
const TRACK_SUFFIX = '@TrackLink';
const EMBED_IMAGE = 'embedImage';

// Load the self-hosted TinyMCE UMD script once.
let tinyPromise = null;
function loadTinyMCE(base) {
  if (window.tinymce) {
    return Promise.resolve(window.tinymce);
  }
  if (tinyPromise) {
    return tinyPromise;
  }

  tinyPromise = new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = `${base}/tinymce.min.js`;
    s.onload = () => resolve(window.tinymce);
    s.onerror = reject;
    document.head.appendChild(s);
  });
  return tinyPromise;
}

class RichtextEditor extends HTMLElement {
  constructor() {
    super();
    this.editor = null;
    this.textarea = null;
    this.pendingValue = null;
  }

  connectedCallback() {
    this._upgradeProperty('value');

    this.textarea = this.querySelector('textarea');
    const initial = this.textarea
      ? this.textarea.value
      : (this.pendingValue ?? this.getAttribute('value') ?? '');
    if (this.textarea) {
      this.textarea.hidden = true;
    }

    const mount = document.createElement('textarea');
    mount.value = initial;
    this.appendChild(mount);
    this._mount = mount;

    const base = this.getAttribute('base') || '/admin/static/tinymce';
    const lang = CONFIG.lang || '';

    loadTinyMCE(base).then((tinymce) => {
      tinymce.init({
        target: mount,
        base_url: base,
        promotion: false,
        branding: false,
        init_instance_callback: (ed) => {
          this.editor = ed;
          if (this.hasAttribute('disabled')) {
            ed.mode.set('readonly');
          }
          // Signal the host that TinyMCE is fully initialized and can be revealed.
          this.dispatchEvent(new CustomEvent('editor-ready', { bubbles: true }));
        },
        urlconverter_callback: (url) => url,

        setup: (editor) => {
          editor.addShortcut('ctrl+s', 'Save content', () => {
            this.dispatchEvent(new CustomEvent('campaign-save', { bubbles: true }));
          });
          editor.addShortcut('f9', 'Preview', () => {
            this.dispatchEvent(new CustomEvent('campaign-preview', { bubbles: true }));
          });

          editor.on('init', () => this._onDialogOpen(editor));

          editor.ui.registry.addButton('html', {
            icon: 'sourcecode',
            tooltip: 'Source code',
            onAction: () => this._openSource(editor),
          });
          editor.ui.registry.addButton('insert-html', {
            icon: 'code-sample',
            tooltip: 'Insert HTML',
            onAction: () => this._openInsertHTML(editor),
          });

          editor.on('change keyup undo redo SetContent', () => this._sync());
          editor.on('CloseWindow', () => {
            editor.selection.getNode().scrollIntoView(false);
          });
        },

        browser_spellcheck: true,
        min_height: 500,
        toolbar_sticky: true,
        entity_encoding: 'raw',
        convert_urls: true,
        relative_urls: false,
        remove_script_host: false,
        extended_valid_elements: 'img[*]',
        plugins: [
          'anchor', 'autoresize', 'autolink', 'charmap', 'emoticons', 'fullscreen',
          'help', 'hr', 'image', 'imagetools', 'link', 'lists', 'paste', 'searchreplace',
          'table', 'visualblocks', 'visualchars', 'wordcount',
        ],
        toolbar: `undo redo | formatselect styleselect fontsizeselect |
                  bold italic underline strikethrough forecolor backcolor subscript superscript |
                  alignleft aligncenter alignright alignjustify |
                  bullist numlist table image insert-html | outdent indent | link hr removeformat |
                  html fullscreen help`,
        fontsize_formats: '10px 11px 12px 14px 15px 16px 18px 24px 36px',
        content_css: false,
        content_style: `
          body { font-family: 'Geist', sans-serif; font-size: 15px; }
          img { max-width: 100%; }
          img.img-float-left { float: left; margin: 0 1em 1em 0; }
          img.img-float-right { float: right; margin: 0 0 1em 1em; }
          a { color: #0055d4; }
          table, td { border-color: #ccc; }
        `,
        language: LANGS[lang] || undefined,
        language_url: LANGS[lang] ? `${base}/lang/${LANGS[lang]}.js` : undefined,

        image_advtab: true,
        image_class_list: [
          { title: 'None', value: '' },
          { title: 'Float left', value: 'img-float-left' },
          { title: 'Float right', value: 'img-float-right' },
        ],
        file_picker_types: 'image',
        file_picker_callback: (cb) => {
          this.dispatchEvent(new CustomEvent('richtext-media', { bubbles: true, detail: { cb } }));
        },
      });
    });
  }

  disconnectedCallback() {
    if (this.editor) {
      // TinyMCE can throw during teardown when its DOM is removed by the framework.
      try { this.editor.remove(); } catch (e) { /* noop */ }
      this.editor = null;
    }
  }

  // Sync editor content back to the textarea and emit `input`.
  _sync() {
    if (!this.editor) {
      return;
    }
    const val = this.editor.getContent();
    if (this.textarea) {
      this.textarea.value = val;
    }
    this.dispatchEvent(new Event('input', { bubbles: true }));
  }

  // Source-code dialog: edit the whole body as HTML, then replace on save.
  _openSource(editor) {
    this._openCodeDialog({
      title: 'Source code',
      initial: editor.getContent(),
      actionLabel: 'Save',
      onAction: (code) => {
        editor.setContent(code);
        this._sync();
      },
    });
  }

  // Insert-HTML dialog: author a snippet and insert it at the caret.
  _openInsertHTML(editor) {
    this._openCodeDialog({
      title: 'Insert HTML',
      initial: '',
      actionLabel: 'Insert',
      onAction: (code) => editor.execCommand('mceInsertContent', false, code),
    });
  }

  // Builds a modal <dialog> hosting a <code-editor> plus a "Format HTML"
  // (beautify) button. `onAction` receives the editor's value on confirm. The
  // dialog is created fresh per open and removed on close so the CodeMirror
  // instance is torn down cleanly.
  _openCodeDialog({
    title, initial, actionLabel, onAction,
  }) {
    const dialog = document.createElement('dialog');
    dialog.className = 'code-source-modal';
    dialog.innerHTML = `
      <div class="dialog-card">
        <header class="dialog-head hstack justify-between">
          <h4 class="dialog-title"></h4>
        </header>
        <section class="dialog-body"></section>
        <footer class="dialog-foot align-right">
          <button type="button" class="outline" data-act="format"></button>
          <button type="button" class="outline" data-act="close"></button>
          <button type="button" data-variant="primary" data-act="action"></button>
        </footer>
      </div>`;

    dialog.querySelector('.dialog-title').textContent = title;

    // The CodeMirror editor, seeded via a nested <textarea> (its documented
    // initial-value channel). <code-editor> is registered by code-editor.js,
    // loaded alongside this component on the page.
    const code = document.createElement('code-editor');
    code.setAttribute('lang', 'html');
    const ta = document.createElement('textarea');
    ta.value = initial;
    code.appendChild(ta);
    dialog.querySelector('.dialog-body').appendChild(code);

    const btnFormat = dialog.querySelector('[data-act="format"]');
    btnFormat.textContent = 'Format HTML';
    btnFormat.addEventListener('click', () => { code.value = beautifyHTML(code.value); });

    const btnClose = dialog.querySelector('[data-act="close"]');
    btnClose.textContent = 'Close';
    btnClose.addEventListener('click', () => dialog.close());

    const btnAction = dialog.querySelector('[data-act="action"]');
    btnAction.textContent = actionLabel;
    btnAction.addEventListener('click', () => {
      onAction(code.value);
      dialog.close();
    });

    dialog.addEventListener('close', () => dialog.remove());
    this.appendChild(dialog);
    dialog.showModal();
  }

  // Injects the "Track link?" / "Embed" checkboxes into the link/image dialogs.
  _onDialogOpen(editor) {
    const ed = editor;
    const oldOpen = ed.windowManager.open;

    ed.windowManager.open = (t, r) => {
      const data = t.initialData || {};
      const isLink = data.url && 'anchor' in data;
      const isImage = data.src && !isLink;
      if (!isLink && !isImage) {
        return oldOpen.call(ed.windowManager, t, r);
      }

      const { onSubmit } = t;
      const checkbox = isLink
        ? { type: 'checkbox', name: TRACK_LINK, label: 'Track link?' }
        : { type: 'checkbox', name: EMBED_IMAGE, label: 'Embed' };
      const spec = { ...t, body: this._withCheckbox(t.body, checkbox) };

      if (isLink) {
        const cleanURL = (data.url.value || '').replace(/@TrackLink$/, '');
        const checked = data.url.value !== cleanURL
          || (!cleanURL && JSON.parse(localStorage.getItem(TRACK_LINK) || 'false'));
        spec.initialData = { ...data, [TRACK_LINK]: checked, url: { ...data.url, value: cleanURL } };
        spec.onSubmit = (api) => {
          const d = api.getData();
          const shouldTrack = Boolean(d[TRACK_LINK]);
          const url = (d.url.value || '').replace(/@TrackLink$/, '');
          localStorage.setItem(TRACK_LINK, JSON.stringify(shouldTrack));
          if (shouldTrack && /^https?:\/\//i.test(url)) {
            api.setData({ url: { ...d.url, value: `${url}${TRACK_SUFFIX}` } });
          }
          onSubmit(api);
        };
      } else {
        const img = this._selectedImage(ed);
        spec.initialData = { ...data, [EMBED_IMAGE]: Boolean(img && img.hasAttribute('data-embed')) };
        spec.onSubmit = (api) => {
          const d = api.getData();
          const shouldEmbed = d[EMBED_IMAGE] === true || d[EMBED_IMAGE] === 'true';
          onSubmit(api);

          const node = (img && ed.getBody().contains(img)) ? img : this._selectedImage(ed);
          if (!node) {
            return;
          }
          if (shouldEmbed) {
            ed.dom.setAttrib(node, 'data-embed', 'true');
          } else {
            node.removeAttribute('data-embed');
          }
          ed.fire('change');
          this._sync();
        };
      }

      return oldOpen.call(ed.windowManager, spec, r);
    };
  }

  _withCheckbox(body, checkbox) {
    if (body.type === 'tabpanel') {
      return {
        ...body,
        tabs: body.tabs.map((tab) => (
          tab.name === 'general' || tab.title === 'General'
            ? { ...tab, items: [...tab.items, checkbox] }
            : tab
        )),
      };
    }
    return { ...body, items: [...body.items, checkbox] };
  }

  _selectedImage(editor) {
    const node = editor.selection.getNode();
    if (!node) {
      return null;
    }
    if (node.nodeName === 'IMG') {
      return node;
    }
    const figure = editor.dom.getParent(node, 'figure.image');
    return figure ? figure.querySelector('img') : null;
  }

  _upgradeProperty(prop) {
    if (Object.prototype.hasOwnProperty.call(this, prop)) {
      const v = this[prop];
      delete this[prop];
      this[prop] = v;
    }
  }

  get value() {
    return this.editor
      ? this.editor.getContent()
      : (this.pendingValue ?? this.getAttribute('value') ?? '');
  }

  set value(val) {
    const v = val == null ? '' : String(val);
    if (!this.editor) {
      this.pendingValue = v;
      if (this._mount) {
        this._mount.value = v;
      }
      return;
    }
    if (v === this.editor.getContent()) {
      return;
    }
    this.editor.setContent(v);
    if (this.textarea) {
      this.textarea.value = v;
    }
  }
}

if (!customElements.get('richtext-editor')) {
  customElements.define('richtext-editor', RichtextEditor);
}
