<template>
  <!-- Two-way Data-Binding -->
  <section class="editor">
    <div class="columns">
      <div class="column is-6">
        <b-field label="Format">
          <b-select v-model="contentType">
            <option :disabled="disabled" name="format" value="richtext" data-cy="check-richtext">
              {{ $t('campaigns.richText') }}
            </option>

            <option :disabled="disabled" name="format" value="html" data-cy="check-html">
              {{ $t('campaigns.rawHTML') }}
            </option>

            <option :disabled="disabled" name="format" value="markdown" data-cy="check-markdown">
              {{ $t('campaigns.markdown') }}
            </option>

            <option :disabled="disabled" name="format" value="plain" data-cy="check-plain">
              {{ $t('campaigns.plainText') }}
            </option>

            <option :disabled="disabled" name="format" value="visual" data-cy="check-visual">
              {{ $t('campaigns.visual') }}
            </option>
          </b-select>
        </b-field>
      </div>
      <div class="column is-6 has-text-right">
        <b-button @click="onTogglePreview" type="is-primary" icon-left="file-find-outline" data-cy="btn-preview">
          {{ $t('campaigns.preview') }} (F9)
        </b-button>
      </div>
    </div>

    <!-- wsywig //-->
    <template v-if="isRichtextReady && computedValue.contentType === 'richtext'">
      <tiny-mce v-model="computedValue.body" :disabled="disabled" :init="richtextConf" />

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
    </template>

    <!-- visual editor //-->
    <visual-editor v-if="computedValue.contentType === 'visual'" :source="computedValue.bodySource" @change="onChangeVisualEditor" />

    <!-- raw html editor //-->
    <html-editor v-if="computedValue.contentType === 'html'" v-model="computedValue.body" />

    <!-- markdown editor //-->
    <markdown-editor v-if="computedValue.contentType === 'markdown'" v-model="computedValue.body" />

    <!-- plain text //-->
    <b-input v-if="computedValue.contentType === 'plain'" v-model="computedValue.body"
      type="textarea" name="content" ref="plainEditor" class="plain-editor" />

    <!-- campaign preview //-->
    <campaign-preview v-if="isPreviewing" @close="onTogglePreview" type="campaign" :id="id" :title="title"
      :content-type="computedValue.contentType" :template-id="computedValue.templateId" :body="computedValue.body" />

    <!-- image picker -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isMediaVisible" :width="900">
      <div class="modal-card content" style="width: auto">
        <section expanded class="modal-card-body">
          <media is-modal @selected="onMediaSelect" />
        </section>
      </div>
    </b-modal>
  </section>
</template>

<script>
import { html as beautifyHTML } from 'js-beautify';
import TurndownService from 'turndown';
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
import Media from '../views/Media.vue';
import CampaignPreview from './CampaignPreview.vue';
import HTMLEditor from './HTMLEditor.vue';
import MarkdownEditor from './MarkdownEditor.vue';
import VisualEditor from './VisualEditor.vue';

const turndown = new TurndownService();

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
    CampaignPreview,
    'html-editor': HTMLEditor,
    'markdown-editor': MarkdownEditor,
    'visual-editor': VisualEditor,
    'tiny-mce': TinyMce,
  },

  props: {
    id: { type: Number, default: 0 },
    title: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
    value: {
      type: Object,
      default: () => ({
        body: '',
        bodySource: null,
        contentType: '',
        templateId: null,
      }),
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

          editor.on('keydown', (e) => {
            if (e.key === 'F9') {
              this.onTogglePreview();
              e.preventDefault();
            }
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
          this.runTinyMceImageCallback = callback;
        },
      };

      this.isRichtextReady = true;
    },

    onContentTypeChange(to, from, prompt) {
      if (this.computedValue.body.trim() === '') {
        return;
      }

      // To avoid prompt loop.
      if (to === this.computedValue.contentType) {
        return;
      }

      if (prompt) {
        // Content isn't empty. Warn.
        this.$utils.confirm(
          this.$t('campaigns.confirmSwitchFormat'),
          () => {
            this.computedValue.contentType = this.contentType;
          },
          () => {
            this.contentType = from;
          },
        );
      } else {
        this.computedValue.contentType = this.contentType;
      }
    },

    convertContentType(to, from) {
      if ((from === 'richtext' || from === 'html') && to === 'plain') {
        // richtext, html => plain

        // Preserve line breaks when converting HTML to plaintext.
        const d = document.createElement('div');
        d.innerHTML = this.beautifyHTML(this.computedValue.body);
        this.$nextTick(() => {
          this.computedValue.body = this.trimLines(d.innerText.trim(), true);
        });
      } else if ((from === 'richtext' || from === 'html') && to === 'markdown') {
        // richtext, html => markdown
        this.computedValue.body = turndown.turndown(this.computedValue.body).replace(/\n\n+/ig, '\n\n');
      } else if (from === 'plain' && (to === 'richtext' || to === 'html')) {
        // plain => richtext, html
        this.computedValue.body = this.computedValue.body.replace(/\n/ig, '<br>\n');
      } else if (from === 'richtext' && to === 'html') {
        // richtext => html
        this.computedValue.body = this.beautifyHTML(this.computedValue.body);
      } else if (from === 'markdown' && (to === 'richtext' || to === 'html')) {
        // markdown => richtext, html.
        this.$api.convertCampaignContent({
          id: 1, body: this.computedValue.body, from, to,
        }).then((data) => {
          this.computedValue.body = this.beautifyHTML(data.trim());
          // Update the HTML editor.
          if (to === 'html') {
            this.updateHTMLEditor();
          }
        });
      }
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
      this.richTextSourceBody = this.computedValue.body;
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
      this.computedValue.body = this.richTextSourceBody;
      window.tinymce.editors[0].setContent(this.computedValue.body);
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

    onTogglePreview() {
      this.isPreviewing = !this.isPreviewing;
    },

    onMediaSelect(media) {
      this.runTinyMceImageCallback(media.url);
    },

    onPreviewShortcut(e) {
      if (e.key === 'F9') {
        this.onTogglePreview();
        e.preventDefault();
      }
    },

    onChangeVisualEditor({ body, source }) {
      this.computedValue.body = body;
      this.computedValue.bodySource = source;
    },

    beautifyHTML(str) {
      // Pad all tags with linebreaks.
      let s = this.trimLines(str.replace(/(<(?!(\/)?a|span)([^>]+)>)/ig, '\n$1\n'), true);
      // Remove extra linebreaks.
      s = s.replace(/\n+/g, '\n');

      return beautifyHTML(s, {
        indent_size: 4,
        indent_char: ' ',
        max_preserve_newlines: 2,
        inline: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'b', 'strong', 'span', 'em', 'i', 'code', 'a'],
      }).trim();
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

    // Set initial content type for the selector.
    this.contentType = this.value.contentType;

    window.addEventListener('keydown', this.onPreviewShortcut);
  },

  beforeDestroy() {
    window.removeEventListener('keydown', this.onPreviewShortcut);
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

  watch: {
    contentType(to, from) {
      this.onContentTypeChange(to, from, true);
    },

    // eslint-disable-next-line func-names
    'computedValue.contentType': function (to, from) {
      this.convertContentType(to, from);
    },
  },
};
</script>
