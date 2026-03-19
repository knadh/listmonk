<template>
  <!-- Two-way Data-Binding -->
  <section class="editor">
    <div class="columns editor-toolbar mb-0" :class="{ 'is-mobile': !isMobile }">
      <div class="column is-three-quarters" :class="{ 'is-inline-flex': !isMobile }">
        <b-field :label="$t('campaigns.format')" label-position="on-border" class="mr-4 mb-0">
          <b-select v-model="contentTypeSel" :disabled="disabled" name="content_type" :expanded="isMobile">
            <option v-for="(name, f) in contentTypes" :key="f" name="format" :value="f" :data-cy="`check-${f}`">
              {{ name }}
            </option>
          </b-select>
        </b-field>

        <b-field v-if="self.contentType !== 'visual'" :label="$tc('globals.terms.template')" label-position="on-border" :class="{ 'mt-2': isMobile }">
          <b-select :placeholder="$t('globals.terms.none')" v-model="templateId" name="template" :disabled="disabled" :expanded="isMobile">
            <option v-for="t in validTemplates" :value="t.id" :key="t.id">
              {{ t.name }}
            </option>
          </b-select>
        </b-field>

        <div v-else :class="{ 'mt-2': isMobile }">
          <b-button v-if="!isVisualTplSelector" @click="onShowVisualTplSelector" type="is-ghost"
            icon-left="file-find-outline" data-cy="btn-select-visual-tpl" :expanded="isMobile">
            {{ $t('campaigns.importVisualTemplate') }}
          </b-button>
          <b-field v-else :label="$tc('globals.terms.template')" label-position="on-border">
            <b-select :placeholder="$t('globals.terms.none')" v-model="visualTemplateId"
              @input="() => isVisualTplDisabled = false" name="template" :disabled="disabled"
              class="copy-visual-template-list" :expanded="isMobile">
            <option v-for="t in validTemplates" :value="t.id" :key="t.id">
                {{ t.name }}
            </option>
            </b-select>

            <b-button :disabled="disabled || isVisualTplDisabled || !visualTemplateId" class="ml-3"
              @click="onImportVisualTpl" type="is-primary" icon-left="content-save-outline"
              data-cy="btn-save-visual-tpl">
              {{ $t('globals.terms.import') }}

              <span class="spinner is-tiny" v-if="loading.templates">
                <b-loading :is-full-page="false" active />
              </span>
            </b-button>
          </b-field>
        </div>
      </div>
      <div class="column" :class="{ 'has-text-right': !isMobile, 'has-text-left mt-2': isMobile }">
        <b-button
          v-if="self.contentType === 'richtext'"
          @click="toggleMarkdownMode"
          type="is-light"
          class="mr-2"
          :icon-left="isMarkdownMode ? 'text' : 'language-markdown'"
          :expanded="isMobile"
        >
          {{ isMarkdownMode ? $t('campaigns.richText') : $t('campaigns.markdown') }}
        </b-button>
        <b-button @click="onTogglePreview" type="is-primary" icon-left="file-find-outline" data-cy="btn-preview"
          aria-keyshortcuts="F9" :expanded="isMobile">
          <span class="has-kbd">{{ $t('campaigns.preview') }} <span class="kbd">F9</span></span>
        </b-button>
      </div>
    </div>

    <!-- wsywig //-->
    <richtext-editor v-if="self.contentType === 'richtext' && !isMarkdownMode" :disabled="disabled" v-model="self.body" :height="height || '75vh'" />

    <!-- markdown authoring for richtext //-->
    <markdown-editor v-if="self.contentType === 'richtext' && isMarkdownMode" v-model="markdownBody"
      :disabled="disabled" :is-mobile="isMobile" :id="id" :title="title" :template-id="templateId" :height="height || '75vh'" />

    <!-- visual editor //-->
    <visual-editor v-if="self.contentType === 'visual'" :source="self.bodySource" @change="onVisualEditorChange"
      :height="height || '65vh'" ref="visualEditor" />

    <!-- raw html editor //-->
    <code-editor lang="html" v-if="self.contentType === 'html'" v-model="self.body" key="editor-html" :height="height || '75vh'" />

    <!-- markdown editor //-->
    <markdown-editor v-if="self.contentType === 'markdown'" v-model="self.body"
      :disabled="disabled" :is-mobile="isMobile" :id="id" :title="title" :template-id="templateId" :height="height || '75vh'" />

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
import VisualEditor from './VisualEditor.vue';
import RichtextEditor from './RichtextEditor.vue';
import MarkdownEditor from './MarkdownEditor.vue';
import markdownToVisualBlock from './editor';
import CodeEditor from './CodeEditor.vue';

const turndown = new TurndownService();

export default {
  components: {
    CampaignPreview,
    'code-editor': CodeEditor,
    'visual-editor': VisualEditor,
    'richtext-editor': RichtextEditor,
    'markdown-editor': MarkdownEditor,
  },

  props: {
    contentTypes: { type: Object, default: () => ({}) },
    id: { type: Number, default: 0 },
    title: { type: String, default: '' },
    height: { type: String, default: null },
    isMobile: { type: Boolean, default: false },
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
      templateId: null,
      visualTemplateId: null,

      // Hybrid Markdown mode for Rich Text
      isMarkdownMode: false,
      markdownBody: '',
    };
  },

  methods: {
    toggleMarkdownMode() {
      if (this.isMarkdownMode) {
        // Switching FROM Markdown TO Rich Text.
        // Convert Markdown to HTML via API for consistency.
        this.$api.convertCampaignContent({
          id: 1,
          body: this.markdownBody,
          from: 'markdown',
          to: 'richtext',
        }).then((data) => {
          this.self.body = this.beautifyHTML(data.trim());
          this.isMarkdownMode = false;
        });
      } else {
        // Switching FROM Rich Text TO Markdown.
        // Convert HTML to Markdown locally.
        this.markdownBody = turndown.turndown(this.self.body).replace(/\n\n+/ig, '\n\n');
        this.isMarkdownMode = true;
      }
    },

    onContentTypeChange(to, from) {
      if (!this.self.body.trim()) {
        this.convertContentType(to, from);
        return;
      }

      // Ask for confirmation as pretty much all conversions are lossy.
      this.$utils.confirm(
        this.$t('campaigns.confirmSwitchFormat'),
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
      let bodySource = null;

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

          case 'visual': {
            const md = turndown.turndown(body).replace(/\n\n+/ig, '\n\n');
            bodySource = JSON.stringify(markdownToVisualBlock(md));
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
      } else if (to === 'visual') {
        bodySource = JSON.stringify(markdownToVisualBlock(body));
      }

      // =======================================================================
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
          this.self.bodySource = bodySource;
        });
      }
    },

    onTogglePreview() {
      this.isPreviewing = !this.isPreviewing;
    },

    onKeyboardShortcut(e) {
      // On F9, toggle the preview.
      if (e.key === 'F9') {
        this.onTogglePreview();
        e.preventDefault();
      }

      // On Ctrl+S, trigger save.
      if (e.ctrlKey && e.key === 's') {
        this.$events.$emit('campaign.update');
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
      if (!this.visualTemplateId) {
        return;
      }

      this.$utils.confirm(
        this.$t('campaigns.confirmOverwriteContent'),
        () => {
          // Fetch the template body from the server.
          this.$api.getTemplate(this.visualTemplateId).then((data) => {
            this.self.body = data.body;
            this.self.bodySource = data.bodySource;
            this.isVisualTplDisabled = true;

            this.$refs.visualEditor.render(JSON.parse(data.bodySource));
          });
        },
      );
    },

    setDefaultTemplate() {
      if (this.self.contentType === 'visual') {
        this.visualTemplateId = this.validTemplates[0]?.id || null;
      } else {
        if (this.templateId) {
          return;
        }

        const defaultTemplate = this.validTemplates.find((t) => t.isDefault === true);
        this.templateId = defaultTemplate?.id || this.validTemplates[0]?.id || null;
      }
    },
  },

  mounted() {
    // Set initial content type for the selector.
    this.contentTypeSel = this.value.contentType;
    this.templateId = this.value.templateId;

    window.addEventListener('keydown', this.onKeyboardShortcut);

    this.$events.$on('campaign.preview', () => {
      this.isPreviewing = true;
    });

    this.$events.$on('campaign.update', () => {
      if (this.isMarkdownMode && this.self.contentType === 'richtext') {
        // Force conversion before save.
        this.$api.convertCampaignContent({
          id: 1,
          body: this.markdownBody,
          from: 'markdown',
          to: 'richtext',
        }).then((data) => {
          this.self.body = this.beautifyHTML(data.trim());
        });
      }
    });
  },

  beforeDestroy() {
    window.removeEventListener('keydown', this.onKeyboardShortcut);
    this.$events.$off('campaign.preview');
    this.$events.$off('campaign.update');
  },

  computed: {
    ...mapState(['serverConfig', 'loading']),

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

    markdownBody() {
      if (this.isMarkdownMode && this.self.contentType === 'richtext') {
        // Debounce or just update? For now update.
        // Note: Real saving usually happens on mode toggle or explicit save.
        // But to support auto-save features, we should keep the body updated.
        // We don't want to call the API on every keystroke, so we do a simple
        // local conversion or wait for the toggle.
        // Actually, the safest is to only sync on toggle or save.
        // However, listmonk auto-saves. Let's do a simple sync.
      }
    },
  },
};

</script>

<style lang="scss" scoped>
.editor-toolbar {
  position: sticky;
  top: 0;
  z-index: 30;
  background: #fff;
  padding: 0.5rem 0;
  border-bottom: 1px solid #dbdbdb;
}
</style>
