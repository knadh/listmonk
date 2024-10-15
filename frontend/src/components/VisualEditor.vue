<template>
  <div class="visual-editor-wrapper">
    <div ref="visualEditor" id="visual-editor" class="visual-editor email-builder-container" />
  </div>
</template>

<script>
import { render, isRendered, setDocument, DEFAULT_SOURCE } from '../email-builder';

export default {
  props: {
    source: { type: String, default: '' },
    height: { type: String, default: 'auto' },
  },

  methods: {
    initEditor() {
      let source = null;
      if (this.$props.source) {
        source = JSON.parse(this.$props.source);
      }

      render('visual-editor', {
        data: source,
        onChange: (data, body) => {
          this.$emit('change', { source: JSON.stringify(data), body });
        },
        height: this.height,
      });
    },
  },

  mounted() {
    this.initEditor();
  },

  watch: {
    source(val) {
      if (isRendered('visual-editor')) {
        if (val) {
          setDocument(JSON.parse(val));
        } else {
          setDocument(DEFAULT_SOURCE);
        }
      } else {
        this.initEditor();
      }
    },
  },
};
</script>

<style lang="css">
  .visual-editor-wrapper {
    width: 100%;
    border: 1px solid #eaeaea;
    max-width: 100vw;
  }

  #visual-editor {
    position: relative;
  }
</style>
