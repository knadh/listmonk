<template>
  <div class="markdown-editor-container" :class="{ 'is-mobile': isMobile, 'is-fullscreen': isFullscreen }" :style="{ height: isFullscreen ? '' : height }">
    <div class="editor-header mb-2" v-if="!disabled">
      <div class="buttons md-toolbar">
        <b-button size="is-small" @click="wrapSelection('**', '**')">
          <span class="material-symbols-outlined">format_bold</span>
        </b-button>
        <b-button size="is-small" @click="wrapSelection('*', '*')">
          <span class="material-symbols-outlined">format_italic</span>
        </b-button>
        <b-button size="is-small" @click="wrapSelection('`', '`')">
          <span class="material-symbols-outlined">code</span>
        </b-button>
        <b-button size="is-small" @click="prefixLines('- ')">
          <span class="material-symbols-outlined">format_list_bulleted</span>
        </b-button>
        <b-button size="is-small" @click="prefixLines('1. ')">
          <span class="material-symbols-outlined">format_list_numbered</span>
        </b-button>
        <b-button size="is-small" @click="wrapSelection('[', '](url)')">
          <span class="material-symbols-outlined">link</span>
        </b-button>
        <b-button size="is-small" @click="isMediaVisible = true">
          <span class="material-symbols-outlined">image</span>
        </b-button>
        <div class="is-divider-vertical mx-1" />
        <b-button size="is-small" @click="isFullscreen = !isFullscreen" :type="isFullscreen ? 'is-primary' : ''">
          <span class="material-symbols-outlined">{{ isFullscreen ? 'fullscreen_exit' : 'fullscreen' }}</span>
        </b-button>
      </div>
      <div v-if="isFullscreen" class="fullscreen-close">
        <b-button size="is-small" type="is-ghost" @click="isFullscreen = false">
          <span class="material-symbols-outlined">close</span>
        </b-button>
      </div>
    </div>

    <div class="editor-layout">
      <div class="editor-pane">
        <code-editor
          ref="codeEditor"
          lang="markdown"
          v-model="internalValue"
          :disabled="disabled"
          class="markdown-code-editor"
        />
      </div>
      <div class="preview-pane" v-if="!isMobile || showPreview">
        <div class="preview-header is-hidden-tablet">
           <b-button size="is-small" icon-left="close" @click="showPreview = false">Close Preview</b-button>
        </div>
        <div class="preview-content">
           <campaign-preview
             v-if="internalValue"
             :is-post="true"
             :id="id"
             :title="title"
             type="campaign"
             content-type="markdown"
             :template-id="templateId"
             :body="internalValue"
             inline
           />
           <div v-else class="has-text-grey has-text-centered mt-6">
             Markdown preview will appear here.
           </div>
        </div>
      </div>
    </div>

    <div class="mobile-toggle is-hidden-tablet" v-if="isMobile">
      <b-button
        expanded
        type="is-primary"
        icon-left="eye"
        @click="showPreview = !showPreview"
      >
        {{ showPreview ? 'Show Editor' : 'Show Preview' }}
      </b-button>
    </div>

    <!-- image picker -->
    <b-modal :aria-modal="true" :active.sync="isMediaVisible" :width="900" class="modal-z-index-high">
      <div class="modal-card content" style="width: auto">
        <section expanded class="modal-card-body">
          <media is-modal @selected="onMediaSelect" />
        </section>
      </div>
    </b-modal>
  </div>
</template>

<script>
import CodeEditor from './CodeEditor.vue';
import CampaignPreview from './CampaignPreview.vue';
import Media from '../views/Media.vue';

export default {
  components: {
    'code-editor': CodeEditor,
    CampaignPreview,
    Media,
  },

  props: {
    value: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
    isMobile: { type: Boolean, default: false },
    id: { type: Number, default: 0 },
    title: { type: String, default: '' },
    templateId: { type: [Number, null], default: null },
    height: { type: String, default: '75vh' },
  },

  data() {
    return {
      internalValue: this.value,
      isMediaVisible: false,
      showPreview: false,
      isFullscreen: false,
    };
  },

  watch: {
    value(val) {
      this.internalValue = val;
    },
    internalValue(val) {
      this.$emit('input', val);
    },
    isFullscreen(val) {
      if (val) {
        document.body.classList.add('has-fullscreen-editor');
      } else {
        document.body.classList.remove('has-fullscreen-editor');
      }
    },
  },

  beforeDestroy() {
    document.body.classList.remove('has-fullscreen-editor');
  },

  methods: {
    onMediaSelect(media) {
      this.insertAtCursor(`![${media.filename}](${media.url})`);
      this.isMediaVisible = false;
    },

    insertAtCursor(text) {
      const { editor } = this.$refs.codeEditor;
      const { state } = editor;
      const { from, to } = state.selection.main;
      editor.dispatch({
        changes: { from, to, insert: text },
        selection: { anchor: from + text.length },
      });
      editor.focus();
    },

    wrapSelection(start, end) {
      const { editor } = this.$refs.codeEditor;
      const { state } = editor;
      const { from, to } = state.selection.main;
      const selection = state.doc.sliceString(from, to);
      const text = `${start}${selection}${end}`;
      editor.dispatch({
        changes: { from, to, insert: text },
        selection: { anchor: from + start.length, head: to + start.length },
      });
      editor.focus();
    },

    prefixLines(prefix) {
      const { editor } = this.$refs.codeEditor;
      const { state } = editor;
      const { from, to } = state.selection.main;
      const lines = state.doc.sliceString(from, to).split('\n');
      const text = lines.map((line) => `${prefix}${line}`).join('\n');
      editor.dispatch({
        changes: { from, to, insert: text },
      });
      editor.focus();
    },
  },
};
</script>

<style lang="scss" scoped>
.markdown-editor-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  border: 1px solid #dbdbdb;
  border-radius: 4px;
  background: #fff;

  .editor-header {
    padding: 0.5rem;
    border-bottom: 1px solid #dbdbdb;
    background: #f5f5f5;
    position: sticky;
    top: 0;
    z-index: 20;
    display: flex;
    justify-content: space-between;
    align-items: center;

    .md-toolbar {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      align-items: center;

      .button {
        width: auto;
        margin-bottom: 0;
        padding: 4px 8px;
      }

      .material-symbols-outlined {
        font-size: 18px;
      }

      .is-divider-vertical {
        display: block;
        width: 1px;
        background-color: #dbdbdb;
        height: 20px;
        margin: 0 4px;
      }
    }
  }

  .editor-layout {
    display: flex;
    flex: 1;
    overflow: hidden;
    min-height: 400px;

    .editor-pane, .preview-pane {
      flex: 1;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }

    .editor-pane {
      border-right: 1px solid #dbdbdb;
      overflow: auto;
    }

    .markdown-code-editor {
      flex: 1;
      height: 100% !important;
      ::v-deep .cm-editor {
        height: 100%;
      }
    }

    .preview-pane {
      background: #fafafa;
      .preview-content {
        flex: 1;
        overflow: auto;
        padding: 1rem;
      }
    }
  }

  &.is-fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 1900;
    border-radius: 0;
    border: none;
    height: 100vh !important;
    max-height: 100vh !important;

    .editor-layout {
      min-height: 0;
      height: calc(100vh - 40px); // Subtract header height approximately.
    }
  }

  &.is-mobile {
    .editor-layout {
      flex-direction: column;
      .editor-pane {
        border-right: none;
        border-bottom: 1px solid #dbdbdb;
      }
      .preview-pane {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 1000;
        background: #fff;
      }
    }
  }

  .mobile-toggle {
    padding: 0.5rem;
    border-top: 1px solid #dbdbdb;
  }
}
</style>
