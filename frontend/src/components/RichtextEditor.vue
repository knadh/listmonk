<template>
  <div class="richtext-editor-container" :class="{ 'is-fullscreen': isFullscreen }">
    <div class="editor-header mb-2" v-if="!disabled">
      <div class="buttons md-toolbar" v-if="editor">
        <!-- History -->
        <b-button size="is-small" @click="editor.chain().focus().undo().run()" :disabled="!editor.can().undo()">
          <span class="material-symbols-outlined">undo</span>
        </b-button>
        <b-button size="is-small" @click="editor.chain().focus().redo().run()" :disabled="!editor.can().redo()">
          <span class="material-symbols-outlined">redo</span>
        </b-button>
        <div class="is-divider-vertical mx-1" />

        <!-- Formatting -->
        <b-button size="is-small" @click="editor.chain().focus().toggleBold().run()" :type="editor.isActive('bold') ? 'is-primary' : ''">
          <span class="material-symbols-outlined">format_bold</span>
        </b-button>
        <b-button size="is-small" @click="editor.chain().focus().toggleItalic().run()" :type="editor.isActive('italic') ? 'is-primary' : ''">
          <span class="material-symbols-outlined">format_italic</span>
        </b-button>
        <b-button size="is-small" @click="editor.chain().focus().toggleStrike().run()" :type="editor.isActive('strike') ? 'is-primary' : ''">
          <span class="material-symbols-outlined">format_strikethrough</span>
        </b-button>
        <b-button size="is-small" @click="editor.chain().focus().toggleCode().run()" :type="editor.isActive('code') ? 'is-primary' : ''">
          <span class="material-symbols-outlined">code</span>
        </b-button>

        <div class="is-divider-vertical mx-1" />

        <!-- Headings -->
        <b-button size="is-small" @click="editor.chain().focus().toggleHeading({ level: 1 }).run()" :type="editor.isActive('heading', { level: 1 }) ? 'is-primary' : ''">
          H1
        </b-button>
        <b-button size="is-small" @click="editor.chain().focus().toggleHeading({ level: 2 }).run()" :type="editor.isActive('heading', { level: 2 }) ? 'is-primary' : ''">
          H2
        </b-button>
        <b-button size="is-small" @click="editor.chain().focus().toggleHeading({ level: 3 }).run()" :type="editor.isActive('heading', { level: 3 }) ? 'is-primary' : ''">
          H3
        </b-button>

        <div class="is-divider-vertical mx-1" />

        <!-- Lists -->
        <b-button size="is-small" @click="editor.chain().focus().toggleBulletList().run()" :type="editor.isActive('bulletList') ? 'is-primary' : ''">
          <span class="material-symbols-outlined">format_list_bulleted</span>
        </b-button>
        <b-button size="is-small" @click="editor.chain().focus().toggleOrderedList().run()" :type="editor.isActive('orderedList') ? 'is-primary' : ''">
          <span class="material-symbols-outlined">format_list_numbered</span>
        </b-button>

        <div class="is-divider-vertical mx-1" />

        <!-- Links & Media -->
        <b-button size="is-small" @click="setLink" :type="editor.isActive('link') ? 'is-primary' : ''">
          <span class="material-symbols-outlined">link</span>
        </b-button>
        <b-button size="is-small" @click="isMediaVisible = true">
          <span class="material-symbols-outlined">image</span>
        </b-button>

        <div class="is-divider-vertical mx-1" />

        <!-- Tables -->
        <b-dropdown role="list" size="is-small">
          <template #trigger>
            <b-button size="is-small">
              <span class="material-symbols-outlined">table_chart</span>
            </b-button>
          </template>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()">Insert Table</b-dropdown-item>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().addColumnBefore().run()" :disabled="!editor.isActive('table')">Add Column Before</b-dropdown-item>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().addColumnAfter().run()" :disabled="!editor.isActive('table')">Add Column After</b-dropdown-item>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().deleteColumn().run()" :disabled="!editor.isActive('table')">Delete Column</b-dropdown-item>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().addRowBefore().run()" :disabled="!editor.isActive('table')">Add Row Before</b-dropdown-item>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().addRowAfter().run()" :disabled="!editor.isActive('table')">Add Row After</b-dropdown-item>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().deleteRow().run()" :disabled="!editor.isActive('table')">Delete Row</b-dropdown-item>
          <b-dropdown-item role="listitem" @click="editor.chain().focus().deleteTable().run()" :disabled="!editor.isActive('table')">Delete Table</b-dropdown-item>
        </b-dropdown>

        <div class="is-divider-vertical mx-1" />

        <!-- Misc -->
        <b-button size="is-small" @click="editor.chain().focus().unsetAllMarks().clearNodes().run()">
          <span class="material-symbols-outlined">format_clear</span>
        </b-button>
        <b-button size="is-small" @click="onRichtextViewSource">
          <span class="material-symbols-outlined">code_blocks</span>
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
      <editor-content :editor="editor" class="tiptap-editor" />
    </div>

    <!-- Source Modal -->
    <b-modal :width="1200" :aria-modal="true" :active.sync="isRichtextSourceVisible">
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">{{ $t('campaigns.sourceCode') || 'Source Code' }}</p>
        </header>
        <section class="modal-card-body">
          <code-editor lang="html" v-model="richTextSourceBody" key="richtext-source" />
        </section>
        <footer class="modal-card-foot is-justify-content-flex-end">
          <b-button @click="onFormatRichtextHTML">
            {{ $t('campaigns.formatHTML') }}
          </b-button>
          <b-button @click="isRichtextSourceVisible = false">
            {{ $t('globals.buttons.close') }}
          </b-button>
          <b-button @click="onSaveRichTextSource" class="is-primary">
            {{ $t('globals.buttons.save') }}
          </b-button>
        </footer>
      </div>
    </b-modal>

    <!-- Image Picker Modal -->
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
import { Editor, EditorContent } from '@tiptap/vue-2';
import StarterKit from '@tiptap/starter-kit';
import Link from '@tiptap/extension-link';
import Image from '@tiptap/extension-image';
import Table from '@tiptap/extension-table';
import TableRow from '@tiptap/extension-table-row';
import TableCell from '@tiptap/extension-table-cell';
import TableHeader from '@tiptap/extension-table-header';
import { html as beautifyHTML } from 'js-beautify';

import CodeEditor from './CodeEditor.vue';
import Media from '../views/Media.vue';

export default {
  components: {
    EditorContent,
    CodeEditor,
    Media,
  },

  props: {
    disabled: { type: Boolean, default: false },
    height: { type: String, default: '75vh' },
    value: { type: String, default: '' },
  },

  data() {
    return {
      editor: null,
      isFullscreen: false,
      isMediaVisible: false,
      isRichtextSourceVisible: false,
      richTextSourceBody: '',
    };
  },

  watch: {
    value(val) {
      const isSame = this.editor.getHTML() === val;
      if (!isSame) {
        this.editor.commands.setContent(val, false);
      }
    },
    isFullscreen(val) {
      if (val) {
        document.body.classList.add('has-fullscreen-editor');
      } else {
        document.body.classList.remove('has-fullscreen-editor');
      }
    },
  },

  mounted() {
    this.editor = new Editor({
      content: this.value,
      editable: !this.disabled,
      extensions: [
        StarterKit,
        Link.configure({
          openOnClick: false,
          HTMLAttributes: {
            class: 'richtext-link',
          },
        }),
        Image.configure({
          HTMLAttributes: {
            class: 'richtext-image',
          },
        }),
        Table.configure({
          resizable: true,
        }),
        TableRow,
        TableHeader,
        TableCell,
      ],
      onUpdate: () => {
        this.$emit('input', this.editor.getHTML());
      },
    });
  },

  beforeDestroy() {
    this.editor.destroy();
    document.body.classList.remove('has-fullscreen-editor');
  },

  methods: {
    setLink() {
      const previousUrl = this.editor.getAttributes('link').href;
      // Simple prompt for now.
      // eslint-disable-next-line no-alert
      const url = window.prompt('URL', previousUrl);

      // cancelled
      if (url === null) {
        return;
      }

      // empty
      if (url === '') {
        this.editor.chain().focus().extendMarkRange('link').unsetLink()
          .run();
        return;
      }

      // update link
      this.editor.chain().focus().extendMarkRange('link').setLink({ href: url })
        .run();
    },

    onMediaSelect(media) {
      this.editor.chain().focus().setImage({ src: media.url, alt: media.filename }).run();
      this.isMediaVisible = false;
    },

    onRichtextViewSource() {
      this.richTextSourceBody = beautifyHTML(this.editor.getHTML());
      this.isRichtextSourceVisible = true;
    },

    onFormatRichtextHTML() {
      this.richTextSourceBody = beautifyHTML(this.richTextSourceBody);
    },

    onSaveRichTextSource() {
      this.editor.commands.setContent(this.richTextSourceBody, true);
      this.isRichtextSourceVisible = false;
    },
  },
};
</script>

<style lang="scss">
.richtext-editor-container {
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
    position: relative;
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
        height: 28px;
        min-width: 28px;
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
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
    min-height: 400px;
    background: #fff;

    .tiptap-editor {
      height: 100%;
      outline: none;

      .ProseMirror {
        height: 100%;
        min-height: 380px;
        outline: none;

        p.is-editor-empty:first-child::before {
          content: attr(data-placeholder);
          float: left;
          color: #adb5bd;
          pointer-events: none;
          height: 0;
        }

        // Tiptap Content Styles
        ul, ol {
          padding: 0 1rem;
          margin: 1rem 0;
          list-style-type: initial;
        }
        ol { list-style-type: decimal; }

        h1, h2, h3, h4, h5, h6 {
          line-height: 1.1;
          margin-top: 1.5rem;
          margin-bottom: 0.5rem;
        }

        code {
          background-color: rgba(#616161, 0.1);
          color: #616161;
        }

        pre {
          background: #0D0D0D;
          color: #FFF;
          font-family: 'JetBrainsMono', monospace;
          padding: 0.75rem 1rem;
          border-radius: 0.5rem;
          code {
            color: inherit;
            padding: 0;
            background: none;
            font-size: 0.8rem;
          }
        }

        img {
          max-width: 100%;
          height: auto;
          &.ProseMirror-selectednode {
            outline: 3px solid #68CEF8;
          }
        }

        blockquote {
          padding-left: 1rem;
          border-left: 3px solid rgba(#0D0D0D, 0.1);
          margin: 1rem 0;
        }

        hr {
          border: none;
          border-top: 2px solid rgba(#0D0D0D, 0.1);
          margin: 2rem 0;
        }

        table {
          border-collapse: collapse;
          table-layout: fixed;
          width: 100%;
          margin: 0;
          overflow: hidden;

          td, th {
            min-width: 1em;
            border: 2px solid #ced4da;
            padding: 3px 5px;
            vertical-align: top;
            box-sizing: border-box;
            position: relative;

            > * {
              margin-bottom: 0;
            }
          }

          th {
            font-weight: bold;
            text-align: left;
            background-color: #f8f9fa;
          }

          .selectedCell:after {
            z-index: 2;
            position: absolute;
            content: "";
            left: 0; right: 0; top: 0; bottom: 0;
            background: rgba(200, 200, 255, 0.4);
            pointer-events: none;
          }

          .column-resize-handle {
            position: absolute;
            right: -2px;
            top: 0;
            bottom: -2px;
            width: 4px;
            background-color: #adf;
            pointer-events: none;
          }
        }

        .tableWrapper {
          overflow-x: auto;
        }

        &.resize-cursor {
          cursor: ew-resize;
          cursor: col-resize;
        }

        a {
          color: #7957d5; // Default primary color
          text-decoration: underline;
        }
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
      height: calc(100vh - 44px);
    }
  }
}
</style>
