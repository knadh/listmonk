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
          </div>
        </b-field>
      </div>
      <div class="column is-6 has-text-right">
          <b-button @click="togglePreview" type="is-primary"
            icon-left="file-find-outline">Preview</b-button>
      </div>
    </div>

    <!-- wsywig //-->
    <quill-editor
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

    <!-- campaign preview //-->
    <campaign-preview v-if="isPreviewing"
      @close="togglePreview"
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

import { quillEditor } from 'vue-quill-editor';
import CodeFlask from 'codeflask';

import CampaignPreview from './CampaignPreview.vue';
import Media from '../views/Media.vue';

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
          toolbar: {
            container: [
              [{ header: [1, 2, 3, false] }],
              ['bold', 'italic', 'underline', 'strike', 'blockquote', 'code'],
              [{ color: [] }, { background: [] }, { size: [] }],
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
              ['clean', 'font'],
            ],

            handlers: {
              image: this.toggleMedia,
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
          .codeflask .codeflask__lines { background: #fafafa; }
        </style>
        <div id="area"></area>
      `;
      this.$refs.htmlEditor.appendChild(el);

      const flask = new CodeFlask(el.shadowRoot.getElementById('area'), {
        language: 'html',
        lineNumbers: true,
        styleParent: el.shadowRoot,
        readonly: this.disabled,
      });

      flask.updateCode(this.form.body);
      flask.onUpdate((b) => {
        this.form.body = b;
        this.$emit('input', { contentType: this.form.format, body: this.form.body });
      });
    },

    togglePreview() {
      this.isPreviewing = !this.isPreviewing;
    },

    toggleMedia() {
      this.lastSel = this.$refs.quill.quill.getSelection();
      this.isMediaVisible = !this.isMediaVisible;
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
  },
};
</script>
