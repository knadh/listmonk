// <code-editor> is a reusable CodeMirror 6 WebComponent.
import { EditorState } from '@codemirror/state';
import {
  EditorView, keymap, highlightActiveLine, lineNumbers, highlightActiveLineGutter,
} from '@codemirror/view';
import { html } from '@codemirror/lang-html';
import { css } from '@codemirror/lang-css';
import { javascript } from '@codemirror/lang-javascript';
import { markdown } from '@codemirror/lang-markdown';
import {
  defaultKeymap, history, historyKeymap, indentWithTab,
} from '@codemirror/commands';
import { defaultHighlightStyle, syntaxHighlighting, bracketMatching } from '@codemirror/language';
import { search, searchKeymap, highlightSelectionMatches } from '@codemirror/search';
import { vsCodeLight } from './editor-theme.js';

const LANGS = {
  html, css, javascript, markdown,
};

class CodeEditor extends HTMLElement {
  constructor() {
    super();
    this.editor = null;
    this.textarea = null;
    this.internalUpdate = false;
  }

  connectedCallback() {
    // Handle if `.value` was assigned before the element was upgraded.
    this._upgradeProperty('value');

    // The (optional) nested textarea with the initial value, which also doubles as the
    // form-submission field. CodeMirror renders into a separate mount div.
    this.textarea = this.querySelector('textarea');
    const initial = this.textarea
      ? this.textarea.value
      : (this.pendingValue ?? this.getAttribute('value') ?? '');
    if (this.textarea) {
      this.textarea.hidden = true;
    }

    const mount = document.createElement('div');
    this.appendChild(mount);

    // Apply .code-editor styles to the <code-editor> tag.
    this.classList.add('code-editor');

    const langFn = LANGS[this.getAttribute('lang')] || html;
    const disabled = this.hasAttribute('disabled');

    const onUpdate = EditorView.updateListener.of((update) => {
      if (!update.docChanged) {
        return;
      }
      this.internalUpdate = true;
      const val = update.state.doc.toString();
      if (this.textarea) {
        this.textarea.value = val;
      }
      this.dispatchEvent(new Event('input', { bubbles: true }));
      this.internalUpdate = false;
    });

    this.editor = new EditorView({
      parent: mount,
      state: EditorState.create({
        doc: initial,
        extensions: [
          langFn(),
          history(),
          highlightActiveLine(),
          bracketMatching(),
          highlightSelectionMatches(),
          lineNumbers(),
          highlightActiveLineGutter(),
          keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap, indentWithTab]),

          // Readonly?
          EditorState.readOnly.of(disabled),
          EditorView.editable.of(!disabled),

          // Syntax highlighting and theme.
          syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
          EditorView.lineWrapping,
          vsCodeLight,

          search({ top: true }),

          onUpdate,
        ],
      }),
    });
  }

  disconnectedCallback() {
    if (this.editor) {
      this.editor.destroy();
      this.editor = null;
    }
  }

  get value() {
    return this.editor
      ? this.editor.state.doc.toString()
      : (this.pendingValue ?? this.getAttribute('value') ?? '');
  }

  set value(val) {
    const v = val == null ? '' : String(val);

    if (!this.editor) {
      this.pendingValue = v;
      return;
    }

    // Ignore self-edit / no change.
    if (this.internalUpdate || v === this.editor.state.doc.toString()) {
      return;
    }

    this.editor.dispatch({
      changes: { from: 0, to: this.editor.state.doc.length, insert: v },
    });
  }

  // Reinstall a property that was set on the instance before the element was upgraded.
  _upgradeProperty(prop) {
    if (Object.prototype.hasOwnProperty.call(this, prop)) {
      const v = this[prop];
      delete this[prop];
      this[prop] = v;
    }
  }
}

if (!customElements.get('code-editor')) {
  customElements.define('code-editor', CodeEditor);
}
