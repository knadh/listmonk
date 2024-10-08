<template>
    <div ref="markdownEditor" id="markdown-editor" class="markdown-editor" />
  </template>

<script>
import 'prismjs/components/prism-markdown';
import CodeFlask from 'codeflask';
import { colors } from '../constants';

export default {
  props: {
    value: { type: String, default: '' },
    language: { type: String, default: 'markdown' },
    disabled: Boolean,
  },

  data() {
    return {
      data: '',
      flask: null,
    };
  },

  methods: {
    initMarkdownEditor(body) {
      // CodeFlask editor is rendered in a shadow DOM tree to keep its styles
      // sandboxed away from the global styles.
      const el = document.createElement('code-flask');
      el.attachShadow({ mode: 'open' });

      el.shadowRoot.innerHTML = `
           <style>
            .codeflask .codeflask__flatten { font-size: 15px; }
            .codeflask .codeflask__lines { background: #f5f5f5; z-index: 10; }
            .codeflask .token.tag { font-weight: bold; color: ${colors.primary}; }
            .codeflask .token.attr-name { color: #111; }
            .codeflask .token.attr-value { color: ${colors.secondary} !important; }
            .codeflask .token.keyword { color: #ff4500; }
            .codeflask .token.comment { color: #6a737d; font-style: italic; }
            .codeflask .token.heading { font-weight: bold; color: #00f; }
            .codeflask .token.important,.token.bold,.token.strong { font-weight: bold; }
            .codeflask .token.em,.token.italic { font-style: italic; }
            .codeflask .token.punctuation { color: #999; }
            .codeflask .token.comment { color: slategray; }
            .codeflask .token.keyword { color: #ff4500; }
            .codeflask .token.string { color: #22863a; }
            .codeflask .token.function { color: #6f42c1; }
            .codeflask .token.url { color: #0366d6; text-decoration: underline; }
          </style>
          <div id="area"></area>
          `;
      this.$refs.markdownEditor.appendChild(el);

      this.flask = new CodeFlask(el.shadowRoot.getElementById('area'), {
        language: this.$props.language || 'markdown',
        lineNumbers: false,
        styleParent: el.shadowRoot,
        readonly: this.disabled,
      });

      this.flask.onUpdate((v) => {
        this.data = v;
        this.$emit('input', v);
      });

      // Set the initial value.
      this.flask.updateCode(body);

      this.$nextTick(() => {
        document.querySelector('code-flask').shadowRoot.querySelector('textarea').focus();
      });
    },
  },

  mounted() {
    this.initMarkdownEditor(this.$props.value || '');
  },

  watch: {
    value(newVal) {
      if (newVal !== this.data) {
        this.flask.updateCode(newVal);
      }
    },
  },
};
</script>
