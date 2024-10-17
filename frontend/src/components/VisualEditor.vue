<template>
  <div class="visual-editor-wrapper">
    <div ref="visualEditor" id="visual-editor" class="visual-editor email-builder-container" />
  </div>
</template>

<script>
import { render } from '../email-builder';

export default {
  props: {
    source: { type: String, default: '' },
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
      });
    },
  },

  mounted() {
    this.initEditor();
  },
};
</script>

<style lang="css">
  .visual-editor-wrapper {
    width: 100%;
    border-top: 1px solid hsl(0, 0%, 86%);
  }

  #visual-editor {
    position: relative;
  }
</style>
