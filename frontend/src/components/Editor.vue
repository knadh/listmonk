<template>
  <!-- Two-way Data-Binding -->
  <section class="editor">
    <div class="columns">
      <div class="column is-three-quarters is-inline-flex">
        <b-field :label="$t('campaigns.format')" label-position="on-border" class="mr-4 mb-0">
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

        <b-field v-if="computedValue.contentType !== 'visual'" :label="$t('globals.terms.baseTemplate')" label-position="on-border">
          <b-select :placeholder="$t('globals.terms.none')" v-model="templateId" name="template" :disabled="disabled">
            <template v-for="t in applicableTemplates">
              <option :value="t.id" :key="t.id">
                {{ t.name }}
              </option>
            </template>
          </b-select>
        </b-field>

        <b-field v-else :label="$t('globals.terms.copyVisualTemplate')" label-position="on-border">
          <b-select :placeholder="$t('globals.terms.none')" v-model="templateId" name="template" :disabled="disabled" class="copy-visual-template-list">
            <option :value="null" key="none" v-if="templateId !== null">
              {{ $t('globals.terms.none') }}
            </option>
            <template v-for="t in applicableTemplates">
              <option v-if="t.type === 'campaign' || t.type === 'campaign_visual'" :value="t.id" :key="t.id">
                {{ t.name }}
              </option>
            </template>
          </b-select>
        </b-field>
      </div>
      <div class="column is- has-text-right">
        <b-button @click="onTogglePreview" type="is-primary" icon-left="file-find-outline" data-cy="btn-preview">
          {{ $t('campaigns.preview') }} (F9)
        </b-button>
      </div>
    </div>

    <!-- wsywig //-->
    <richtext-editor v-if="computedValue.contentType === 'richtext'" v-model="computedValue.body" />

    <!-- visual editor //-->
    <visual-editor v-if="computedValue.contentType === 'visual'" :source="computedValue.bodySource" @change="onChangeVisualEditor" height="65vh" />

    <!-- raw html editor //-->
    <html-editor v-if="computedValue.contentType === 'html'" v-model="computedValue.body" />

    <!-- markdown editor //-->
    <markdown-editor v-if="computedValue.contentType === 'markdown'" v-model="computedValue.body" />

    <!-- plain text //-->
    <b-input v-if="computedValue.contentType === 'plain'" v-model="computedValue.body"
      type="textarea" name="content" ref="plainEditor" class="plain-editor" />

    <!-- campaign preview //-->
    <campaign-preview v-if="isPreviewing" is-post @close="onTogglePreview" type="campaign" :id="id" :title="title"
      :content-type="computedValue.contentType" :template-id="templateId" :body="computedValue.body" />
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
      contentType: this.$props.value.contentType,
      templateId: '',
    };
  },

  methods: {
    onContentTypeChange(to, from) {
      if (this.computedValue.body?.trim() === '') {
        this.computedValue.contentType = this.contentType;
        return;
      }

      // To avoid prompt loop.
      if (to === this.computedValue.contentType) {
        return;
      }

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
    },

    convertContentType(to, from) {
      let body;
      if ((from === 'richtext' || from === 'html') && to === 'plain') {
        // richtext, html => plain

        // Preserve line breaks when converting HTML to plaintext.
        const d = document.createElement('div');
        d.innerHTML = this.beautifyHTML(this.computedValue.body);
        body = this.trimLines(d.innerText.trim(), true);
      } else if ((from === 'richtext' || from === 'html') && to === 'markdown') {
        // richtext, html => markdown
        body = turndown.turndown(this.computedValue.body).replace(/\n\n+/ig, '\n\n');
      } else if (from === 'plain' && (to === 'richtext' || to === 'html')) {
        // plain => richtext, html
        body = this.computedValue.body.replace(/\n/ig, '<br>\n');
      } else if (from === 'richtext' && to === 'html') {
        // richtext => html
        body = this.beautifyHTML(this.computedValue.body);
      } else if (from === 'markdown' && (to === 'richtext' || to === 'html')) {
        // markdown => richtext, html.
        this.$api.convertCampaignContent({
          id: 1, body: this.computedValue.body, from, to,
        }).then((data) => {
          this.$nextTick(() => {
            this.computedValue.body = this.beautifyHTML(data.trim());
          });
        });
      }

      // Update the current body.
      this.$nextTick(() => {
        this.computedValue.body = body;
      });

      // Reset template ID only if its converted to or from visual template.
      if (to === 'visual' || from === 'visual') {
        this.templateId = null;
        this.computedValue.templateId = null;
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
    // Set initial content type for the selector.
    this.contentType = this.value.contentType;
    this.templateId = this.value.templateId;

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

    applicableTemplates() {
      if (this.computedValue.contentType === 'visual') {
        return this.templates.filter((t) => t.type === 'campaign_visual');
      } else {
        return this.templates.filter((t) => t.type === 'campaign');
      }
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

    applicableTemplates(to) {
      if (this.computedValue.contentType !== 'visual') {
        const ctps = this.templates.filter((t) => t.type === 'campaign')
        if (!ctps.find(t => t.id === this.templateId)) {
            const defaultTemplate = ctps.find(t => t.isDefault === true);
            this.templateId = defaultTemplate?.id || ctps[0]?.id || null;
        }
      }
    },

    templateId(to, from) {
      if (this.computedValue.templateId === to) {
        return;
      }

      if (this.computedValue.contentType === 'visual') {
        this.$utils.confirm(
          this.$t('campaigns.confirmApplyVisualTemplate'),
          () => {
            this.computedValue.templateId = to;

            if (!to) {
              this.computedValue.body = '';
              this.computedValue.bodySource = null;
            } else {
              this.templates.forEach((t) => {
                if (t.id === to) {
                  this.computedValue.body = t.body;
                  this.computedValue.bodySource = t.bodySource;
                }
              });
            }
          },
          () => {
            this.templateId = from;
          },
        );
      } else {
        this.computedValue.templateId = to;
      }
    },
  },
};
</script>
