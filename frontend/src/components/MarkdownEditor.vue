<template>
  <div ref="markdownEditor" class="markdown-editor">
    <code-input
      ref="editor"
      :value="value"
      @input="handleInput"
      language="markdown"
      :data-readonly="disabled"
      spellcheck="false"
    />
  </div>
</template>

<script>
// import 'code-input';  // Import the web component
// import 'prismjs/components/prism-markdown';
import Prism from 'prismjs';

export default {
  props: {
    value: { type: String, default: '' },
    language: { type: String, default: 'markdown' },
    disabled: Boolean,
  },

  data() {
    return {
      data: '',
    };
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

  watch: {
    value(newVal) {
      if (newVal !== this.data) {
        this.data = newVal;
      }
    },
  },
};
</script>

<style>
.markdown-editor {
  width: 100%;
  position: relative;
}

/* Base editor styles */
.markdown-editor textarea {
  font-size: 15px;
  min-height: 200px;
  width: 100%;
  height: 100%;
  border: none;
  border-radius: 2px;
  padding: 8px;
  box-sizing: border-box; /* Keep padding within the width/height */
  resize: none; /* Optional: Prevent resizing */
}

/* Markdown syntax highlighting */
.markdown-editor .token {
  color: var(--primary-color, #0066cc);
}

.markdown-editor .token.heading {
  font-weight: bold;
}

.markdown-editor .token.important,
.markdown-editor .token.bold,
.markdown-editor .token.strong {
  font-weight: bold;
}

.markdown-editor .token.em,
.markdown-editor .token.italic {
  font-style: italic;
}

.markdown-editor .token.comment {
  color: slategray;
}

.markdown-editor .token.url {
  color: var(--primary-color, #0066cc);
  text-decoration: underline;
}

/* Focus state */
.markdown-editor textarea:focus {
  outline: none;
  border-color: var(--primary-color, #0066cc);
}
</style>
