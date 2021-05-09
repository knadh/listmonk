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
    <quill-editor
      :class="{'fullscreen': isEditorFullscreen}"
      v-if="form.format === 'richtext'"
      v-model="form.body"
      ref="quill"
      :options="options"
      :disabled="disabled"
      :placeholder="$t('campaigns.contentHelp')"
      @change="onEditorChange($event)"
      @ready="onEditorReady($event)"
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
import 'quill/dist/quill.snow.css';
import 'quill/dist/quill.core.css';

import { quillEditor, Quill } from 'vue-quill-editor';
import CodeFlask from 'codeflask';
import TurndownService from 'turndown';

import CampaignPreview from './CampaignPreview.vue';
import Media from '../views/Media.vue';

// Setup Quill to use inline CSS style attributes instead of classes.
Quill.register(Quill.import('attributors/attribute/direction'), true);
Quill.register(Quill.import('attributors/style/align'), true);
Quill.register(Quill.import('attributors/style/background'), true);
Quill.register(Quill.import('attributors/style/color'), true);
Quill.register(Quill.import('formats/indent'), true);

const quillFontSizes = Quill.import('attributors/style/size');
quillFontSizes.whitelist = ['11px', '13px', '22px', '32px'];
Quill.register(quillFontSizes, true);

// Sanitize {{ TrackLink "xxx" }} quotes to backticks.
const regLink = new RegExp(/{{(\s+)?TrackLink(\s+)?"(.+?)"(\s+)?}}/);
const Link = Quill.import('formats/link');
Link.sanitize = (l) => l.replace(regLink, '{{ TrackLink `$3`}}');

const turndown = new TurndownService();

// Custom class to override the default indent behaviour to get inline CSS
// style instead of classes.
class IndentAttributor extends Quill.import('parchment').Attributor.Style {
  multiplier = 30;

  add(node, value) {
    return super.add(node, `${value * this.multiplier}px`);
  }

  value(node) {
    return parseFloat(super.value(node)) / this.multiplier || undefined;
  }
}

export default {
  components: {
    Media,
    CampaignPreview,
    quillEditor,
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

      // Quill editor options.
      options: {
        placeholder: this.$t('campaigns.contentHelp'),
        modules: {
          keyboard: {
            bindings: {
              esc: {
                key: 27,
                handler: () => {
                  this.onToggleFullscreen(true);
                },
              },
            },
          },
          toolbar: {
            container: [
              [{ header: [1, 2, 3, false] }],
              ['bold', 'italic', 'underline', 'strike', 'blockquote', 'code'],
              [{ color: [] }, { background: [] }, { size: quillFontSizes.whitelist }],
              [
                { list: 'ordered' },
                { list: 'bullet' },
                { indent: '-1' },
                { indent: '+1' },
              ],
              [
                { align: '' },
                { align: 'center' },
                { align: 'right' },
                { align: 'justify' },
              ],
              ['link', 'image'],
              ['clean', 'fullscreen'],
            ],

            handlers: {
              image: this.onToggleMedia,
              fullscreen: () => this.onToggleFullscreen(false),
            },
          },
        },
      },

      // HTML editor.
      flask: null,
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

    onEditorReady() {
      this.isReady = true;

      // Hack to focus the editor on page load.
      this.$nextTick(() => {
        window.setTimeout(() => this.$refs.quill.quill.focus(), 100);
      });
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

    onToggleMedia() {
      this.lastSel = this.$refs.quill.quill.getSelection();
      this.isMediaVisible = !this.isMediaVisible;
    },

    onToggleFullscreen(onlyMinimize) {
      if (onlyMinimize) {
        if (!this.isEditorFullscreen) {
          return;
        }
      }
      this.isEditorFullscreen = !this.isEditorFullscreen;
    },

    onMediaSelect(m) {
      this.$refs.quill.quill.insertEmbed(this.lastSel.index || 0, 'image', m.url);
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

  mounted() {
    // Initialize the Quill indentation plugin.
    const levels = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
    const multiplier = 30;
    const indentStyle = new IndentAttributor('indent', 'margin-left', {
      scope: Quill.import('parchment').Scope.BLOCK,
      whitelist: levels.map((value) => `${value * multiplier}px`),
    });

    Quill.register(indentStyle);
  },
};
</script>
