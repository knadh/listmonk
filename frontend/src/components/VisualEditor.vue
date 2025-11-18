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

    render(source) {
      const iframe = this.$refs.visualEditor;

      // If the editor is not-rendered, render it the first time. This can happen
      // on first loads and importing an email template via render().
      const em = iframe.contentWindow.EmailBuilder;
      if (!em || !em.isRendered('visual-editor-container')) {
        iframe.contentWindow.EmailBuilder.render('visual-editor-container', {
          data: {},
          onChange: (data, body) => {
            // Hack to fix quotes in Go {{ templating }} in the HTML body.
            const tpl = body.replace(/\{\{[^}]*\}\}/g, (match) => match.replace(/&quot;/g, '"'));
            this.$emit('change', { source: JSON.stringify(data), body: tpl });
          },
        });
      }

      if (!source) {
        return;
      }

      // setDocument() will trigger onChange() that produces both bodySource and body (HTML).
      // On init, the `data: source` above sets the content in the editor, but doesn't trigger
      // onChange(), which is required to set the source+HTML state in the parent for preview to work.
      // Couldn't figure out if there was an on load/on init event etc. in email-builder, so brute force it
      // with a timer.
      let n = 10;
      const timer = window.setInterval(() => {
        const container = iframe.contentWindow.document.getElementById('visual-editor-container');
        if (container && container.hasChildNodes()) {
          em.resetDocument(source);
          window.clearInterval(timer);
          return;
        }

        n += 1;
        if (n > 10) {
          window.clearInterval(timer);
        }
      }, 100);
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
        let source = null;
        if (this.$props.source) {
          source = JSON.parse(this.$props.source);
        }

        this.render(source);
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
