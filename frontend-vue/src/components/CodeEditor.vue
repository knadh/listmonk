<template>
  <div ref="editor" class="code-editor" />
</template>

<script>
import { EditorState } from '@codemirror/state';
import {
  EditorView, keymap, highlightActiveLine, lineNumbers, highlightActiveLineGutter,
} from '@codemirror/view';
import { markdown } from '@codemirror/lang-markdown';
import { javascript } from '@codemirror/lang-javascript';
import { css } from '@codemirror/lang-css';
import { html } from '@codemirror/lang-html';
import {
  defaultKeymap, history, historyKeymap, indentWithTab,
} from '@codemirror/commands';
import { defaultHighlightStyle, syntaxHighlighting, bracketMatching } from '@codemirror/language';
import { search, searchKeymap, highlightSelectionMatches } from '@codemirror/search';
import { vsCodeLight } from './editor-theme';

export default {
  props: {
    value: { type: String, default: '' },
    lang: { type: String, default: 'html' },
    disabled: Boolean,
  },

  data() {
    return {
      data: '',
      editor: null,
      internalUpdate: false,
    };
  },

  methods: {
  },

  mounted() {
    const onUpdate = EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        this.internalUpdate = true;
        this.$emit('input', update.state.doc.toString());
      }
    });

    // Set the chosen language.
    let langs = [];
    switch (this.lang) {
      case 'html':
        langs = [html()];
        break;
      case 'css':
        langs = [css()];
        break;
      case 'javascript':
        langs = [javascript()];
        break;
      case 'markdown':
        langs = [markdown()];
        break;
      default:
        langs = [html()];
    }

    // Prepare the full config.
    const stateCfg = EditorState.create({
      // Initial value.
      doc: this.value,

      extensions: [
        EditorView.baseTheme({}),
        ...langs,
        history(),
        highlightActiveLine(),
        bracketMatching(),
        highlightSelectionMatches(),
        lineNumbers(),
        highlightActiveLineGutter(),
        keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap, indentWithTab]),

        // Readonly?
        EditorState.readOnly.of(this.disabled),
        EditorView.editable.of(!this.disabled),

        // Syntax highlighting and theme.
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        EditorView.lineWrapping,

        vsCodeLight,

        search({
          top: true, // Places the search panel at the top of the editor
        }),

        // On content change.
        onUpdate,
      ],
    });

    // Create the editor.
    this.editor = new EditorView({
      state: stateCfg,
      parent: this.$refs.editor,
    });

    this.$nextTick(() => {
      window.setTimeout(() => {
        this.editor.focus();
      }, 100);
    });
  },

  beforeDestroy() {
    if (this.editor) {
      this.editor.destroy();
    }
  },

  watch: {
    value(val) {
      if (!this.internalUpdate) {
        this.editor.dispatch({
          changes: { from: 0, to: this.editor.state.doc.length, insert: val },
        });
        this.internalUpdate = false;
      }
    },
  },
};
</script>
