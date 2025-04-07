<template>
  <!-- Two-way Data-Binding -->
  <section class="editor">
    <div class="columns">
      <div class="column is-three-quarters is-inline-flex">
        <b-field :label="$t('campaigns.format')" label-position="on-border" class="mr-4 mb-0">
          <b-select v-model="contentTypeSel">
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

        <b-field v-if="self.contentType !== 'visual'" :label="$tc('globals.terms.template')" label-position="on-border">
          <b-select :placeholder="$t('globals.terms.none')" v-model="templateId" name="template" :disabled="disabled">
            <template v-for="t in validTemplates">
              <option :value="t.id" :key="t.id">
                {{ t.name }}
              </option>
            </template>
          </b-select>
        </b-field>

        <div v-else>
          <b-button v-if="!isVisualTplSelector" @click="onShowVisualTplSelector" type="is-ghost"
            icon-left="file-find-outline" data-cy="btn-select-visual-tpl">
            {{ $t('campaigns.importVisualTemplate') }}
          </b-button>

          <b-field v-else :label="$tc('globals.terms.template')" label-position="on-border">
            <b-select :placeholder="$t('globals.terms.none')" v-model="visualTemplateId"
              @input="() => isVisualTplDisabled = false" name="template" :disabled="disabled"
              class="copy-visual-template-list">
              <template v-for="t in validTemplates">
                <option :value="t.id" :key="t.id">
                  {{ t.name }}
                </option>
              </template>
            </b-select>

            <b-button :disabled="disabled || isVisualTplDisabled" class="ml-3" @click="onImportVisualTpl"
              type="is-primary" icon-left="content-save-outline" data-cy="btn-save-visual-tpl">
              {{ $t('globals.terms.import') }}
            </b-button>
          </b-field>
        </div>
      </div>
      <div class="column is- has-text-right">
        <b-button @click="onTogglePreview" type="is-primary" icon-left="file-find-outline" data-cy="btn-preview">
          {{ $t('campaigns.preview') }} (F9)
        </b-button>
      </div>
    </div>

    <!-- wsywig //-->
    <richtext-editor v-if="self.contentType === 'richtext'" v-model="self.body" />

    <!-- visual editor //-->
    <visual-editor v-if="self.contentType === 'visual'" :source="self.bodySource" @change="onVisualEditorChange"
      height="65vh" />

    <!-- raw html editor //-->
    <html-editor v-if="self.contentType === 'html'" v-model="self.body" />

    <!-- markdown editor //-->
    <markdown-editor v-if="self.contentType === 'markdown'" v-model="self.body" />

    <!-- plain text //-->
    <b-input v-if="self.contentType === 'plain'" v-model="self.body" type="textarea" name="content" ref="plainEditor"
      class="plain-editor" />

    <!-- campaign preview //-->
    <campaign-preview v-if="isPreviewing" is-post @close="onTogglePreview" type="campaign" :id="id" :title="title"
      :content-type="self.contentType" :template-id="templateId" :body="self.body" />
  </section>
</template>

<script>
import { html as beautifyHTML } from 'js-beautify';
import TurndownService from 'turndown';
import { mapState } from 'vuex';

import CampaignPreview from './CampaignPreview.vue';
import HTMLEditor from './HTMLEditor.vue';
import MarkdownEditor from './MarkdownEditor.vue';
import VisualEditor from './VisualEditor.vue';
import RichtextEditor from './RichtextEditor.vue';

const turndown = new TurndownService();

export default {
  components: {
    CampaignPreview,
    'html-editor': HTMLEditor,
    'markdown-editor': MarkdownEditor,
    'visual-editor': VisualEditor,
    'richtext-editor': RichtextEditor,
  },

  props: {
    id: { type: Number, default: 0 },
    title: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
    templates: { type: Array, default: null },

    // value is provided by the parent component.
    // Throught the editor, `this.self` (a mutable clone of `value`) is used,
    // instead of `this.value` directly.
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
      isVisualTplSelector: false,
      isVisualTplDisabled: false,
      contentTypeSel: this.$props.value.contentType,
      templateId: '',
      visualTemplateId: '',
    };
  },

  methods: {
    onContentTypeChange(to, from) {
      // Ask for confirmation as pretty much all conversions are lossy.
      const msgKey = to === 'visual' ? 'campaigns.confirmOverwriteContent' : 'campaigns.confirmSwitchFormat';
      this.$utils.confirm(
        this.$t(msgKey),
        () => {
          this.convertContentType(to, from);
        },
        () => {
          // Cancelled. Reset the <select> to the last value.
          this.contentTypeSel = from;
        },
      );
    },

    convertContentType(to, from) {
      let body = this.self.body ?? '';

      // Skip UI update (markdown => richtext, html requires a backenbd call).
      let skip = false;

      // If `from` is HTML content, strip out `<body>..` etc. and keep the beautified HTML.
      let isHTML = false;
      if (from === 'richtext' || from === 'html' || from === 'visual') {
        const d = document.createElement('div');
        d.innerHTML = body;
        body = this.beautifyHTML(d.innerHTML.trim());
        isHTML = true;
      }

      // HTML => Non-HTML.
      if (isHTML) {
        switch (to) {
          case 'plain': {
            const d = document.createElement('div');
            d.innerHTML = body;
            body = this.trimLines(d.innerText.trim(), true);
            break;
          }

          case 'markdown': {
            body = turndown.turndown(body).replace(/\n\n+/ig, '\n\n');
            break;
          }

          default:
            // Switching between HTML formats, no need to do anything further
            // as body is already beautified.
            // richtext|html => visual, the contents are simply lost.
            break;
        }

        // Markdown to HTML requires a backend call.
      } else if (from === 'markdown' && (to === 'richtext' || to === 'html')) {
        skip = true;
        this.$api.convertCampaignContent({
          id: 1, body, from, to,
        }).then((data) => {
          this.$nextTick(() => {
            // Both type + body should be updated in one cycle to avoid firing
            // multiple events.
            this.self.contentType = to;
            this.self.body = this.beautifyHTML(data.trim());
          });
        });

        // Plain to an HTML type, change plain line breaks to HTML breaks.
      } else if (from === 'plain' && (to === 'richtext' || to === 'html')) {
        body = body.replace(/\n/ig, '<br>\n');
      }

      // =======================================================================
      // If the target is visual, empty the visual editor's block content source.
      if (to !== 'visual') {
        this.self.bodySource = null;
      }

      // Reset the campaign template ID if its converted to or from visual template.
      if (to === 'visual' || from === 'visual') {
        this.templateId = null;
        this.self.templateId = null;
      }

      // =======================================================================
      // Apply the conversion on the editor UI.
      if (!skip) {
        this.$nextTick(() => {
          // Both type + body should be updated in one cycle to avoid firing
          // multiple events.
          this.self.contentType = to;
          this.self.body = body;
        });
      }
    },

    onTogglePreview() {
      this.isPreviewing = !this.isPreviewing;
    },

    onPreviewShortcut(e) {
      if (e.key === 'F9') {
        this.onTogglePreview();
        e.preventDefault();
      }
    },

    onVisualEditorChange({ body, source }) {
      this.self.body = body;
      this.self.bodySource = source;
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

    onShowVisualTplSelector() {
      this.isVisualTplSelector = true;
      this.setDefaultTemplate();
    },

    onImportVisualTpl() {
      this.$utils.confirm(
        this.$t('campaigns.confirmOverwriteContent'),
        () => {
          let found = false;
          this.templates.forEach((t) => {
            if (t.id === this.visualTemplateId) {
              found = true;
              this.self.body = t.body;
              this.self.bodySource = t.bodySource;

              // Deplay update so that applied template is propogated to visual editor
              // and it doesn't enable the apply button again. Delay here is arbitrary.
              setTimeout(() => {
                this.isVisualTplDisabled = true;
              }, 250);
            }
          });

          if (!found) {
            this.self.body = '';
            this.self.bodySource = null;

            // Deplay update so that applied template is propogated to visual editor
            // and it doesn't enable the apply button again. Delay here is arbitrary.
            setTimeout(() => {
              this.isVisualTplDisabled = true;
            }, 250);
          }
        },
      );
    },

    setDefaultTemplate() {
      if (this.self.contentType === 'visual') {
        this.visualTemplateId = this.validTemplates[0]?.id || null;
      } else {
        const defaultTemplate = this.validTemplates.find((t) => t.isDefault === true);
        this.templateId = defaultTemplate?.id || this.validTemplates[0]?.id || null;
      }
    },
  },

  mounted() {
    // Set initial content type for the selector.
    this.contentTypeSel = this.value.contentType;
    this.templateId = this.value.templateId;

    window.addEventListener('keydown', this.onPreviewShortcut);
  },

  beforeDestroy() {
    window.removeEventListener('keydown', this.onPreviewShortcut);
  },

  computed: {
    ...mapState(['serverConfig']),

    // This is a clone of the incoming `value` prop that's mutated here.
    self: {
      get() {
        return this.value;
      },

      // Any change to the local copy, emit it to the parent.
      set(val) {
        this.$emit('input', val);
      },
    },

    // Returns the list of valid (visual vs. normal) templates for the template dropdown.
    validTemplates() {
      const typ = this.self.contentType === 'visual' ? 'campaign_visual' : 'campaign';
      return this.templates.filter((t) => (t.type === typ));
    },
  },

  watch: {
    validTemplates() {
      // When the filtered list of validTemplates changes (visual vs. regular),
      // select the appropriate 'default' in the template select list.
      this.setDefaultTemplate();
    },

    contentTypeSel(to, from) {
      // Show the conversion prompt if the value in the dropdown isn't the same
      // as the current selection. This happens when eg: contentTypeSel = html -> visual happens
      // in the selector, the prompt is shown, and Cancel is clicked,
      // at which point, contentTypeSel = html again, which triggers this event.
      if (from !== to && to !== this.self.contentType) {
        this.onContentTypeChange(to, from);
      }
    },

    templateId(to) {
      if (this.self.templateId === to) {
        return;
      }

      this.self.templateId = to;
    },
  },
};
</script>
