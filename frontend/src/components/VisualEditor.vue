<template>
  <div class="visual-editor-wrapper">
    <iframe ref="visualEditor" id="visual-editor" class="visual-editor email-builder-container"
      title="Visual email editor" />

    <!-- image picker -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isMediaVisible" :width="900">
      <div class="modal-card content" style="width: auto">
        <section expanded class="modal-card-body">
          <media is-modal @selected="onMediaSelect" />
        </section>
      </div>
    </b-modal>
  </div>
</template>

<script>
import Media from '../views/Media.vue';

export default {
  components: {
    Media,
  },

  props: {
    source: { type: String, default: '' },
    height: { type: String, default: 'auto' },
  },

  data() {
    return {
      isMediaVisible: false,
    };
  },

  methods: {
    initEditor() {
      let source = null;
      if (this.$props.source) {
        source = JSON.parse(this.$props.source);
      }

      const iframe = this.$refs.visualEditor;
      iframe.contentWindow.EmailBuilder.render('visual-editor-container', {
        data: source,
        onChange: (data, body) => {
          this.$emit('change', { source: JSON.stringify(data), body });
        },
      });
    },

    loadScript() {
      return new Promise((resolve, reject) => {
        const iframe = this.$refs.visualEditor;
        if (iframe.contentWindow.EmailBuilder) {
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

        // Append script to iframe's head
        iframe.contentDocument.head.appendChild(script);
      });
    },

    // Inject media URL into the image URL input field in the visual edior sidebar.
    onMediaSelect(media) {
      const iframe = this.$refs.visualEditor;
      const input = iframe.contentDocument.querySelector('.image-url input');
      if (input) {
        const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
          window.HTMLInputElement.prototype,
          'value',
        ).set;
        nativeInputValueSetter.call(input, media.url);

        const inputEvent = new Event('input', { bubbles: true });
        input.dispatchEvent(inputEvent);
      }
    },

    // Observe DOM changes in the iframe to inject media selector
    // into the image URL input fields.
    onSidebarMount(msg) {
      if (!msg.data) {
        return;
      }

      if (msg.data === 'visualeditor.select-media') {
        this.isMediaVisible = true;
      }
    },
  },

  mounted() {
    // Initialize iframe content
    const iframe = this.$refs.visualEditor;
    iframe.style.height = this.height;

    // Set basic iframe HTML structure
    iframe.srcdoc = `
      <!DOCTYPE html>
      <html>
        <head>
          <style>
            body { margin: 0; padding: 0; }
            #visual-editor-container { width: 100%; height: 100%; }
          </style>
        </head>
        <body>
          <div id="visual-editor-container"></div>
        </body>
      </html>
    `;

    iframe.onload = () => {
      this.loadScript().then(() => {
        this.initEditor();
      }).catch((error) => {
        /* eslint-disable-next-line no-console */
        console.error('Failed to load email-builer script:', error);
      });
    };

    window.addEventListener('message', this.onSidebarMount, false);
  },

  unmounted() {
    window.removeEventListener('message', this.onSidebarMount, false);
  },

  watch: {
    source(val) {
      const iframe = this.$refs.visualEditor;
      if (iframe.contentWindow.EmailBuilder?.isRendered('visual-editor-container')) {
        if (val) {
          iframe.contentWindow.EmailBuilder.setDocument(JSON.parse(val));
        } else {
          iframe.contentWindow.EmailBuilder.setDocument(iframe.contentWindow.EmailBuilder.DEFAULT_SOURCE);
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
  border: none;
  width: 100%;
  min-height: 500px;
}
</style>
