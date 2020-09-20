<template>
  <!-- Two-way Data-Binding -->
  <section class="editor">
    <div class="columns">
      <div class="column is-6">
        <b-field label="Format">
          <div>
            <b-radio v-model="form.radioFormat"
              @input="onChangeFormat" :disabled="disabled" name="format"
              native-value="richtext">Rich text</b-radio>
            <b-radio v-model="form.radioFormat"
              @input="onChangeFormat" :disabled="disabled" name="format"
              native-value="html">Raw HTML</b-radio>
            <b-radio v-model="form.radioFormat"
              @input="onChangeFormat" :disabled="disabled" name="format"
              native-value="plain">Plain text</b-radio>
          </div>
        </b-field>
      </div>
      <div class="column is-6 has-text-right">
          <b-button @click="onTogglePreview" type="is-primary"
            icon-left="file-find-outline">Preview</b-button>
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
      placeholder="Content here"
      @change="onEditorChange($event)"
      @ready="onEditorReady($event)"
    />

    <!-- raw html editor //-->
    <div v-if="form.format === 'html'"
      ref="htmlEditor" id="html-editor" class="html-editor"></div>

    <!-- plain text editor //-->
    <b-input v-if="form.format === 'plain'" v-model="form.body" @input="onEditorChange"
      type="textarea" ref="plainEditor" class="plain-editor" />

    <!-- campaign preview //-->
    <campaign-preview v-if="isPreviewing"
      @close="onTogglePreview"
      type='campaign'
      :id='id'
      :title='title'
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
        placeholder: 'Content here',
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
    };
  },

  methods: {
    onChangeFormat(format) {
      this.$utils.confirm(
        'The content may lose some formatting. Are you sure?',
        () => {
          this.form.format = format;
          this.onEditorChange();
        },
        () => {
          // On cancel, undo the radio selection.
          this.form.radioFormat = format === 'richtext' ? 'html' : 'richtext';
        },
      );
    },

    onEditorReady() {
      // Hack to focus the editor on page load.
      this.$nextTick(() => {
        window.setTimeout(() => this.$refs.quill.quill.focus(), 100);
      });
    },

    onEditorChange() {
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

      const flask = new CodeFlask(el.shadowRoot.getElementById('area'), {
        language: 'html',
        lineNumbers: false,
        styleParent: el.shadowRoot,
        readonly: this.disabled,
      });

      flask.updateCode(this.form.body);
      flask.onUpdate((b) => {
        this.form.body = b;
        this.$emit('input', { contentType: this.form.format, body: this.form.body });
      });
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

      // Trigger the change event so that the body and content type
      // are propagated to the parent on first load.
      this.onEditorChange();
    },

    body(b) {
      this.form.body = b;
    },

    htmlFormat(f) {
      if (f !== 'html') {
        return;
      }

      this.$nextTick(() => {
        this.initHTMLEditor();
      });
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
