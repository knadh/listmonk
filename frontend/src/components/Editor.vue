<template>
  <!-- Two-way Data-Binding -->
  <section class="editor">
    <div class="columns">
      <div class="column is-6">
        <b-field label="Format">
          <div>
            <b-radio v-model="form.radioFormat"
              @input="onChangeFormat" :disabled="disabled" name="format"
              native-value="richtext"
              data-cy="check-richtext">{{ $t('campaigns.richText') }}</b-radio>
            <b-radio v-model="form.radioFormat"
              @input="onChangeFormat" :disabled="disabled" name="format"
              native-value="html"
              data-cy="check-html">{{ $t('campaigns.rawHTML') }}</b-radio>
            <b-radio v-model="form.radioFormat"
              @input="onChangeFormat" :disabled="disabled" name="format"
              native-value="markdown"
              data-cy="check-markdown">{{ $t('campaigns.markdown') }}</b-radio>
            <b-radio v-model="form.radioFormat"
              @input="onChangeFormat" :disabled="disabled" name="format"
              native-value="plain"
              data-cy="check-plain">{{ $t('campaigns.plainText') }}</b-radio>
          </div>
        </b-field>
      </div>
      <div class="column is-6 has-text-right">
          <b-button @click="onTogglePreview" type="is-primary"
            icon-left="file-find-outline">{{ $t('campaigns.preview') }}</b-button>
      </div>
    </div>

    <!-- wsywig //-->
    <tiny-mce
      v-model="form.body"
      v-if="form.format === 'richtext'"
      :disabled="disabled"
      :init="tinyMceOptions"
      @init="() => isReady = true"
    />

    <!-- raw html editor //-->
    <div v-if="form.format === 'html'"
      ref="htmlEditor" id="html-editor" class="html-editor"></div>

    <!-- plain text / markdown editor //-->
    <b-input v-if="form.format === 'plain' || form.format === 'markdown'"
      v-model="form.body" @input="onEditorChange"
      type="textarea" name="content" ref="plainEditor" class="plain-editor" />

    <!-- campaign preview //-->
    <campaign-preview v-if="isPreviewing"
      @close="onTogglePreview"
      type="campaign"
      :id="id"
      :title="title"
      :contentType="form.format"
      :body="form.body"></campaign-preview>

    <!-- image picker -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isMediaVisible" :width="900">
      <div class="modal-card content" style="width: auto">
        <section expanded class="modal-card-body">
          <media isModal @selected="onMediaSelect" />
        </section>
      </div>
    </b-modal>
  </section>
</template>

<script>
import CodeFlask from 'codeflask';
import TurndownService from 'turndown';

import 'tinymce';
import 'tinymce/icons/default';
import 'tinymce/themes/silver';
import 'tinymce/skins/ui/oxide/skin.css';

import 'tinymce/plugins/charmap';
import 'tinymce/plugins/code';
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

import TinyMce from '@tinymce/tinymce-vue';
import CampaignPreview from './CampaignPreview.vue';
import Media from '../views/Media.vue';

const turndown = new TurndownService();
import { colors } from '../constants';

export default {
  components: {
    Media,
    CampaignPreview,
    TinyMce,
  },

  props: {
    id: Number,
    title: String,
    body: String,
    contentType: String,
    disabled: Boolean,
  },

  data() {
    return {
      isPreviewing: false,
      isMediaVisible: false,
      isEditorFullscreen: false,
      isReady: false,
      form: {
        body: '',
        format: this.contentType,

        // Model bound to the checkboxes. This changes on click of the radio,
        // but is reverted by the change handler if the user cancels the
        // conversion warning. This is used to set the value of form.format
        // that the editor uses to render content.
        radioFormat: this.contentType,
      },

      // Last position of the cursor in the editor before the media popup
      // was opened. This is used to insert media on selection from the poup
      // where the caret may be lost.
      lastSel: null,

      // HTML editor.
      flask: null,

      // TODO: Possibility to overwrite this config from an external source for easy customization
      tinyMceOptions: {
        // TODO: To use `language`, you have to download your desired language pack here:
        //   https://www.tiny.cloud/get-tiny/language-packages/, then somehow copy it into /frontend/js/langs/
        language: this.$t('settings.general.language'),
        height: 500,
        plugins: [
          'charmap',
          'code',
          'emoticons',
          'fullscreen',
          'help',
          'hr',
          'image',
          'imagetools',
          'link',
          'lists',
          'paste',
          'searchreplace',
          'table',
          'visualblocks',
          'visualchars',
          'wordcount',
        ],
        toolbar: 'undo redo | formatselect styleselect fontsizeselect | '
          + 'bold italic underline strikethrough forecolor backcolor subscript superscript | '
          + 'alignleft aligncenter alignright alignjustify | '
          + 'bullist numlist table image | outdent indent | link hr removeformat | '
          + 'code fullscreen help',
        skin: false,
        content_css: false,
        // TODO: Get the `content_style` from an external source (to match the template styles)
        content_style: 'body { font-family: \'Helvetica Neue\', \'Segoe UI\', Helvetica, '
          + 'sans-serif; font-size: 15px; line-height: 26px; color: #444; }'
          + 'img { max-width: 100%; }'
          + 'a { color: #0055d4; }'
          + 'a:hover { color: #111; }',
        file_picker_types: 'image',
        file_picker_callback: (callback) => {
          this.isMediaVisible = true;
          this.runTinyMceImageCallback = callback;
        },
      },
    };
  },

  methods: {
    onChangeFormat(format) {
      this.$utils.confirm(
        this.$t('campaigns.confirmSwitchFormat'),
        () => {
          this.form.format = format;
          this.onEditorChange();
        },
        () => {
          // On cancel, undo the radio selection.
          this.form.radioFormat = this.form.format;
        },
      );
    },

    onEditorChange() {
      if (!this.isReady) {
        return;
      }

      // The parent's v-model gets { contentType, body }.
      this.$emit('input', { contentType: this.form.format, body: this.form.body });
    },

    initHTMLEditor() {
      // CodeFlask editor is rendered in a shadow DOM tree to keep its styles
      // sandboxed away from the global styles.
      const el = document.createElement('code-flask');
      el.attachShadow({ mode: 'open' });
      el.shadowRoot.innerHTML = `
        <style>
          .codeflask .codeflask__flatten { font-size: 15px; }
          .codeflask .codeflask__lines { background: #fafafa; z-index: 10; }
          .codeflask .token.tag { font-weight: bold; }
          .codeflask .token.attr-name { color: #111; }
          .codeflask .token.attr-value { color: ${colors.primary} !important; }
        </style>
        <div id="area"></area>
      `;
      this.$refs.htmlEditor.appendChild(el);

      this.flask = new CodeFlask(el.shadowRoot.getElementById('area'), {
        language: 'html',
        lineNumbers: false,
        styleParent: el.shadowRoot,
        readonly: this.disabled,
      });
      this.flask.onUpdate((b) => {
        this.form.body = b;
        this.$emit('input', { contentType: this.form.format, body: this.form.body });
      });

      this.updateHTMLEditor();
      this.isReady = true;
    },

    updateHTMLEditor() {
      this.flask.updateCode(this.form.body);
    },

    onTogglePreview() {
      this.isPreviewing = !this.isPreviewing;
    },

    onMediaSelect(media) {
      this.runTinyMceImageCallback(media.url);
    },

    beautifyHTML(str) {
      const div = document.createElement('div');
      div.innerHTML = str.trim();
      return this.formatHTMLNode(div, 0).innerHTML;
    },

    formatHTMLNode(node, level) {
      const lvl = level + 1;
      const indentBefore = new Array(lvl + 1).join('  ');
      const indentAfter = new Array(lvl - 1).join('  ');
      let textNode = null;

      for (let i = 0; i < node.children.length; i += 1) {
        textNode = document.createTextNode(`\n${indentBefore}`);
        node.insertBefore(textNode, node.children[i]);

        this.formatHTMLNode(node.children[i], lvl);
        if (node.lastElementChild === node.children[i]) {
          textNode = document.createTextNode(`\n${indentAfter}`);
          node.appendChild(textNode);
        }
      }

      return node;
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

  computed: {
    htmlFormat() {
      return this.form.format;
    },
  },

  watch: {
    // Capture contentType and body passed from the parent as props.
    contentType(f) {
      this.form.format = f;
      this.form.radioFormat = f;

      if (f === 'plain' || f === 'markdown') {
        this.isReady = true;
      }

      // Trigger the change event so that the body and content type
      // are propagated to the parent on first load.
      this.onEditorChange();
    },

    body(b) {
      this.form.body = b;
      this.onEditorChange();
    },

    htmlFormat(to, from) {
      // On switch to HTML, initialize the HTML editor.
      if (to === 'html') {
        this.$nextTick(() => {
          this.initHTMLEditor();
        });
      }

      if ((from === 'richtext' || from === 'html') && to === 'plain') {
        // richtext, html => plain

        // Preserve line breaks when converting HTML to plaintext. Quill produces
        // HTML without any linebreaks.
        const d = document.createElement('div');
        d.innerHTML = this.beautifyHTML(this.form.body);
        this.form.body = this.trimLines(d.innerText.trim(), true);
      } else if ((from === 'richtext' || from === 'html') && to === 'markdown') {
        // richtext, html => markdown
        this.form.body = turndown.turndown(this.form.body).replace(/\n\n+/ig, '\n\n');
      } else if (from === 'plain' && (to === 'richtext' || to === 'html')) {
        // plain => richtext, html
        this.form.body = this.form.body.replace(/\n/ig, '<br>\n');
      } else if (from === 'richtext' && to === 'html') {
        // richtext => html
        this.form.body = this.trimLines(this.beautifyHTML(this.form.body), false);
      } else if (from === 'markdown' && (to === 'richtext' || to === 'html')) {
        // markdown => richtext, html.
        this.$api.convertCampaignContent({
          id: 1, body: this.form.body, from, to,
        }).then((data) => {
          this.form.body = this.beautifyHTML(data.trim());
          // Update the HTML editor.
          if (to === 'html') {
            this.updateHTMLEditor();
          }
        });
      }

      this.onEditorChange();
    },
  },
};
</script>
