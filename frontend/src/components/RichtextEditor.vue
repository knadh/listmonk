<template>
  <div class="richtext-editor" v-if="isRichtextReady">
    <tiny-mce v-model="computedValue" :disabled="disabled" :init="richtextConf" />

    <b-modal scroll="keep" :width="1200" :aria-modal="true" :active.sync="isRichtextSourceVisible">
      <div>
        <section expanded class="modal-card-body preview">
          <html-editor v-model="richTextSourceBody" />
        </section>
        <footer class="modal-card-foot has-text-right">
          <b-button @click="onFormatRichtextHTML">
            {{ $t('campaigns.formatHTML') }}
          </b-button>
          <b-button @click="() => { this.isRichtextSourceVisible = false; }">
            {{ $t('globals.buttons.close') }}
          </b-button>
          <b-button @click="onSaveRichTextSource" class="is-primary">
            {{ $t('globals.buttons.save') }}
          </b-button>
        </footer>
      </div>
    </b-modal>

    <b-modal scroll="keep" :width="750" :aria-modal="true" :active.sync="isInsertHTMLVisible">
      <div>
        <section expanded class="modal-card-body preview">
          <html-editor v-model="insertHTMLSnippet" />
        </section>
        <footer class="modal-card-foot has-text-right">
          <b-button @click="onFormatRichtextHTMLSnippet">
            {{ $t('campaigns.formatHTML') }}
          </b-button>
          <b-button @click="() => { this.isInsertHTMLVisible = false; }">
            {{ $t('globals.buttons.close') }}
          </b-button>
          <b-button @click="onInsertHTML" class="is-primary">
            {{ $t('globals.buttons.insert') }}
          </b-button>
        </footer>
      </div>
    </b-modal>

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
import { html } from 'js-beautify';
import { mapState } from 'vuex';

import TinyMce from '@tinymce/tinymce-vue';
import 'tinymce';
import 'tinymce/icons/default';
import 'tinymce/plugins/anchor';
import 'tinymce/plugins/autolink';
import 'tinymce/plugins/autoresize';
import 'tinymce/plugins/charmap';
import 'tinymce/plugins/colorpicker';
import 'tinymce/plugins/contextmenu';
import 'tinymce/plugins/emoticons';
import 'tinymce/plugins/emoticons/js/emojis';
import 'tinymce/plugins/fullscreen';
import 'tinymce/plugins/help';
import 'tinymce/plugins/hr';
import 'tinymce/plugins/image';
import 'tinymce/plugins/imagetools';
import 'tinymce/plugins/link';
import 'tinymce/plugins/lists';
import 'tinymce/plugins/paste';
import 'tinymce/plugins/searchreplace';
import 'tinymce/plugins/table';
import 'tinymce/plugins/textcolor';
import 'tinymce/plugins/visualblocks';
import 'tinymce/plugins/visualchars';
import 'tinymce/plugins/wordcount';
import 'tinymce/skins/ui/oxide/skin.css';
import 'tinymce/themes/silver';

import { colors, uris } from '../constants';
import HTMLEditor from './HTMLEditor.vue';
import Media from '../views/Media.vue';

// Map of listmonk language codes to corresponding TinyMCE language files.
const LANGS = {
  'cs-cz': 'cs',
  de: 'de',
  es: 'es_419',
  fr: 'fr_FR',
  it: 'it_IT',
  pl: 'pl',
  pt: 'pt_PT',
  'pt-BR': 'pt_BR',
  ro: 'ro',
  tr: 'tr',
};

export default {
  components: {
    Media,
    'tiny-mce': TinyMce,
    'html-editor': HTMLEditor,
  },

  props: {
    disabled: { type: Boolean, default: false },
    value: {
      type: String,
      default: '',
    },
  },

  data() {
    return {
      isPreviewing: false,
      isMediaVisible: false,
      isReady: false,
      isRichtextReady: false,
      isRichtextSourceVisible: false,
      isInsertHTMLVisible: false,
      insertHTMLSnippet: '',
      isTrackLink: false,
      richtextConf: {},
      richTextSourceBody: '',
      contentType: '',
    };
  },

  methods: {
    initRichtextEditor() {
      const { lang } = this.serverConfig;

      this.richtextConf = {
        init_instance_callback: () => { this.isReady = true; },
        urlconverter_callback: this.onEditorURLConvert,

        setup: (editor) => {
          editor.addShortcut('ctrl+s', 'Save content', () => {
            this.$events.$emit('campaign.update', {});
          });
          editor.addShortcut('f9', 'Preview', () => {
            this.$events.$emit('campaign.preview', {});
          });

          editor.on('init', () => {
            editor.focus();
            this.onEditorDialogOpen(editor);
          });

          // Custom HTML editor.
          editor.ui.registry.addButton('html', {
            icon: 'sourcecode',
            tooltip: 'Source code',
            onAction: this.onRichtextViewSource,
          });

          editor.ui.registry.addButton('insert-html', {
            icon: 'code-sample',
            tooltip: 'Insert HTML',
            onAction: this.onOpenInsertHTML,
          });

          editor.on('CloseWindow', () => {
            editor.selection.getNode().scrollIntoView(false);
          });
        },

        browser_spellcheck: true,
        min_height: 500,
        toolbar_sticky: true,
        entity_encoding: 'raw',
        convert_urls: true,
        plugins: [
          'anchor', 'autoresize', 'autolink', 'charmap', 'emoticons', 'fullscreen',
          'help', 'hr', 'image', 'imagetools', 'link', 'lists', 'paste', 'searchreplace',
          'table', 'visualblocks', 'visualchars', 'wordcount',
        ],
        toolbar: `undo redo | formatselect styleselect fontsizeselect |
                  bold italic underline strikethrough forecolor backcolor subscript superscript |
                  alignleft aligncenter alignright alignjustify |
                  bullist numlist table image insert-html | outdent indent | link hr removeformat |
                  html fullscreen help`,
        fontsize_formats: '10px 11px 12px 14px 15px 16px 18px 24px 36px',
        skin: false,
        content_css: false,
        content_style: `
          body { font-family: 'Inter', sans-serif; font-size: 15px; }
          img { max-width: 100%; }
          a { color: ${colors.primary}; }
          table, td { border-color: #ccc;}
        `,

        language: LANGS[lang] || null,
        language_url: LANGS[lang] ? `${uris.static}/tinymce/lang/${LANGS[lang]}.js` : null,

        file_picker_types: 'image',
        file_picker_callback: (callback) => {
          this.isMediaVisible = true;
          this.imageCallack = callback;
        },
      };

      this.isRichtextReady = true;
    },

    onEditorURLConvert(url) {
      let u = url;
      if (this.isTrackLink) {
        u = `${u}@TrackLink`;
      }

      this.isTrackLink = false;
      return u;
    },

    onRichtextViewSource() {
      this.richTextSourceBody = this.computedValue;
      this.isRichtextSourceVisible = true;
    },

    onOpenInsertHTML() {
      this.isInsertHTMLVisible = true;
    },

    onInsertHTML() {
      this.isInsertHTMLVisible = false;
      window.tinymce.editors[0].execCommand('mceInsertContent', false, this.insertHTMLSnippet);
      this.insertHTMLSnippet = '';
    },

    onFormatRichtextHTML() {
      this.richTextSourceBody = this.beautifyHTML(this.richTextSourceBody);
    },

    onFormatRichtextHTMLSnippet() {
      this.insertHTMLSnippet = this.beautifyHTML(this.insertHTMLSnippet);
    },

    onSaveRichTextSource() {
      this.computedValue = this.richTextSourceBody;
      window.tinymce.editors[0].setContent(this.computedValue);
      this.richTextSourceBody = '';
      this.isRichtextSourceVisible = false;
    },

    onEditorDialogOpen(editor) {
      const ed = editor;
      const oldEd = ed.windowManager.open;
      const self = this;

      ed.windowManager.open = (t, r) => {
        const isOK = t.initialData && 'url' in t.initialData && 'anchor' in t.initialData;

        // Not the link modal.
        if (!isOK) {
          return oldEd.apply(this, [t, r]);
        }

        // If an existing link is being edited, check for the tracking flag `@TrackLink` at the end
        // of the url. Remove that from the URL and instead check the checkbox.
        let checked = false;
        if (!t.initialData.link !== '') {
          const t2 = t;
          const url = t2.initialData.url.value.replace(/@TrackLink$/, '');

          if (t2.initialData.url.value !== url) {
            t2.initialData.url.value = url;
            checked = true;
          }
        }

        // Execute the modal.
        const modal = oldEd.apply(this, [t, r]);

        // Is it the link dialog?
        if (isOK) {
          // Insert tracking checkbox.
          const c = document.createElement('input');
          c.setAttribute('type', 'checkbox');

          if (checked) {
            c.setAttribute('checked', checked);
          }

          // Store the checkbox's state in the Vue instance to pick up from
          // the TinyMCE link conversion callback.
          c.onchange = (e) => {
            self.isTrackLink = e.target.checked;
          };

          const l = document.createElement('label');
          l.appendChild(c);
          l.appendChild(document.createTextNode('Track link?'));
          l.classList.add('tox-label', 'tox-track-link');

          document.querySelector('.tox-form__controls-h-stack .tox-control-wrap').appendChild(l);
        }
        return modal;
      };
    },

    onMediaSelect(media) {
      this.imageCallack(media.url);
    },

    beautifyHTML(str) {
      // Pad all tags with linebreaks.
      let s = this.trimLines(str.replace(/(<(?!(\/)?a|span)([^>]+)>)/ig, '\n$1\n'), true);
      // Remove extra linebreaks.
      s = s.replace(/\n+/g, '\n');

      try {
        s = html(s).trim();
      } catch (error) {
        // eslint-disable-next-line no-console
        console.log('error formatting HTML', error);
      }

      return s;
    },

    trimLines(str, removeEmptyLines) {
      const out = str.split('\n');
      for (let i = 0; i < out.length; i += 1) {
        const line = out[i].trim();
        if (removeEmptyLines) {
          out[i] = line;
        } else if (line === '') {
          out[i] = '';
        }
      }

      return out.join('\n').replace(/\n\s*\n\s*\n/g, '\n\n');
    },
  },

  mounted() {
    this.initRichtextEditor();
  },

  computed: {
    ...mapState(['serverConfig']),

    computedValue: {
      get() {
        return this.value;
      },
      set(newValue) {
        this.$emit('input', newValue);
      },
    },
  },
};
</script>
