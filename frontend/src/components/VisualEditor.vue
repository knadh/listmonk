<template>
  <div class="visual-editor-wrapper">
    <div ref="visualEditor" id="visual-editor" class="visual-editor email-builder-container" />

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

    onMediaSelect(media) {
      const input = document.querySelector('.image-url input');
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
  },

  mounted() {
    this.loadScript().then(() => {
      this.initEditor();
    }).catch((error) => {
      // eslint-disable-next-line no-console
      console.error('Failed to load email-builder script:', error);
    });

    // Select a stable parent element
    const stableParent = document.querySelector('.visual-editor');

    // Observe mutations to handle dynamically added elements
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        mutation.addedNodes.forEach((node) => {
          node.querySelectorAll('.image-url').forEach((img) => {
            // Create anchor tag
            const anchor = document.createElement('a');
            anchor.href = '#';
            anchor.className = 'open-media-selector';
            anchor.textContent = 'Select Image';
            anchor.style.marginTop = '5px';
            anchor.addEventListener('click', (e) => {
              e.preventDefault();
              this.isMediaVisible = true;
            });
            if (img.parentNode) {
              img.parentNode.insertBefore(anchor, img.nextSibling);
            }
          });
        });
      });
    });

    // Start observing
    observer.observe(stableParent, {
      childList: true,
      subtree: true,
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
