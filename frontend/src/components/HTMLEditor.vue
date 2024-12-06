<template>
  <div ref="htmlEditor" class="html-editor">
    <code-input
      ref="editor"
      :value="value"
      @input="handleInput"
      language="html"
      :data-readonly="disabled"
      spellcheck="false"
    />
  </div>
</template>

<script>
import Prism from 'prismjs';

export default {
  props: {
    value: { type: String, default: '' },
    language: { type: String, default: 'html' },
    disabled: Boolean,
  },

  methods: {
    handleInput(event) {
      this.$emit('input', event.target.value);
    },

    initializeEditor() {
      // const textarea = this.$refs.editor;
      // textarea.setAttribute('is', 'code-input');
      // textarea.setAttribute('data-language', this.language);

      // Register Prism for syntax highlighting if needed
      if (window.codeInput) {
        window.codeInput.registerTemplate(
          'syntax-highlighted',
          window.codeInput.templates.prism(Prism, []),
        );
        // window.codeInput.setDefaultTemplate('syntax-highlighted');
      }
    },
  },

  mounted() {
    this.initializeEditor();
  },
};
</script>

<style>
/* Hide the non-editable preview content */
.code-input pre[aria-hidden="true"] {
  display: none !important;
}

/* Additional styling */
.html-editor {
  width: 100%;
  position: relative;
}

.html-editor textarea {
  font-size: 15px;
  min-height: 200px;
  width: 100%;
  padding: 8px;
  border: none;
  resize: none;
}

.token.tag { font-weight: bold; }
.token.attr-name { color: #111; }
.token.attr-value { color: #0066cc; }

.html-editor textarea:focus {
  outline: none;
  border-color: #0066cc;
}
</style>
