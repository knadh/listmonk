<template>
  <div class="visual-editor-wrapper">
    <div ref="visualEditor" id="visual-editor" class="visual-editor email-builder-container" />
  </div>
</template>

<script>
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

      window.EmailBuilder.render('visual-editor', {
        data: source,
        onChange: (data, body) => {
          this.$emit('change', { source: JSON.stringify(data), body });
        },
        height: this.height,
      });
    },

    loadScript() {
      return new Promise((resolve, reject) => {
        if (window.EmailBuilder) {
          resolve();
          return;
        }

        const script = document.createElement('script');
        script.id = 'email-builder-script';
        script.src = '/admin/static/email-builder/email-builder.umd.js';
        script.onload = () => {
          resolve();
        };
        script.onerror = reject;
        document.head.appendChild(script);
      });
    },
  },

  mounted() {
    this.loadScript().then(() => {
      this.initEditor();
    }).catch((error) => {
      // eslint-disable-next-line no-console
      console.error('Failed to load email-builder script:', error);
    });
  },

  watch: {
    source(val) {
      if (window.EmailBuilder?.isRendered('visual-editor')) {
        if (val) {
          window.EmailBuilder.setDocument(JSON.parse(val));
        } else {
          window.EmailBuilder.setDocument(window.EmailBuilder.DEFAULT_SOURCE);
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
