<template>
  <div class="tiptap-editor">
    <div v-if="editor" class="editor-menu">
      <b-dropdown aria-role="list">
        <template #trigger>
          <b-button
            size="is-small"
            :label="currentHeadingLabel"
            :icon-left="currentHeadingIcon"
          />
        </template>

        <b-dropdown-item
          @click="editor.chain().focus().toggleHeading({ level: 1 }).run()"
          aria-role="listitem"
        >
          Heading 1
        </b-dropdown-item>
        <b-dropdown-item
          @click="editor.chain().focus().toggleHeading({ level: 2 }).run()"
          aria-role="listitem"
        >
          Heading 2
        </b-dropdown-item>
        <b-dropdown-item
          @click="editor.chain().focus().toggleHeading({ level: 3 }).run()"
          aria-role="listitem"
        >
          Heading 3
        </b-dropdown-item>
        <b-dropdown-item @click="editor.chain().focus().setParagraph().run()" aria-role="listitem">
          Normal
        </b-dropdown-item>
      </b-dropdown>
      <div v-for="(actions, groupName) in actionGroups" :key="groupName">
        <b-button
          v-for="action in actions"
          :key="action.label"
          :icon-left="action.icon"
          @click="action.onClick()"
          size="is-small"
          :type="editor.isActive(action.isActiveCondition) ? 'is-primary is-light':''"
        >
          <template v-if="!action.icon">
            {{ action.label }}
          </template>
        </b-button>
      </div>
    </div>
    <editor-content :editor="editor" class="tiptap-editor__content p-2" />
  </div>
</template>

<script>
import { Editor, EditorContent } from '@tiptap/vue-2';
import StarterKit from '@tiptap/starter-kit';
import TextAlign from '@tiptap/extension-text-align';

export default {
  components: {
    EditorContent,
  },

  model: {
    prop: 'modelValue',
    event: 'update:modelValue',
  },

  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },

  data() {
    return {
      editor: null,
      actionGroups: {
        textActions: [
          {
            onClick: () => this.editor.chain().focus().toggleBold().run(),
            label: 'bold',
            icon: 'format-bold',
            isActiveCondition: 'bold',
          },
          {
            onClick: () => this.editor.chain().focus().toggleItalic().run(),
            label: 'italic',
            icon: 'format-italic',
            isActiveCondition: 'italic',
          },
          {
            onClick: () => this.editor.chain().focus().editor.toggleUnderline().run(),
            label: 'underline',
            icon: 'format-underline',
            isActiveCondition: 'underline',
          },
          {
            onClick: () => this.editor.chain().focus().toggleStrike().run(),
            label: 'strike',
            icon: 'format-strikethrough-variant',
            isActiveCondition: 'strike',
          },
          {
            onClick: () => this.editor.chain().focus().toggleBlockquote().run(),
            label: 'blockquote',
            icon: 'format-quote-close',
            isActiveCondition: 'blockquote',
          },
          {
            onClick: () => this.editor.chain().focus().toggleCodeBlock().run(),
            label: 'code block',
            icon: 'code',
            isActiveCondition: 'codeBlock',
          },
        ],
        blockActions: [
          {
            onClick: () => this.editor.chain().focus().toggleBulletList().run(),
            label: 'bullet list',
            icon: 'format-list-bulleted-square',
            isActiveCondition: 'bulletList',
          },
          {
            onClick: () => this.editor.chain().focus().toggleOrderedList().run(),
            label: 'ordered list',
            icon: 'format-list-numbered',
            isActiveCondition: 'orderedList',
          },
        ],
        alignActions: [
          {
            onClick: () => this.editor.chain().focus().setTextAlign('left').run(),
            label: 'Align left',
            icon: 'format-align-left',
            isActiveCondition: { textAlign: 'left' },
          },
          {
            onClick: () => this.editor.chain().focus().setTextAlign('center').run(),
            label: 'Align center',
            icon: 'format-align-center',
            isActiveCondition: { textAlign: 'center' },
          },
          {
            onClick: () => this.editor.chain().focus().setTextAlign('right').run(),
            label: 'Align right',
            icon: 'format-align-right',
            isActiveCondition: { textAlign: 'right' },
          },
          {
            onClick: () => this.editor.chain().focus().setTextAlign('justify').run(),
            label: 'Align left',
            icon: 'format-align-justify',
            isActiveCondition: { textAlign: 'justify' },
          },
        ],
      },
    };
  },

  computed: {
    currentHeadingLabel() {
      if (this.editor.isActive('heading')) {
        if (this.editor.isActive('heading', { level: 1 })) {
          return 'Heading 1';
        }
        if (this.editor.isActive('heading', { level: 2 })) {
          return 'Heading 2';
        }
        if (this.editor.isActive('heading', { level: 3 })) {
          return 'Heading 3';
        }
      }
      return 'Paragraph';
    },
    currentHeadingIcon() {
      if (this.editor.isActive('heading')) {
        if (this.editor.isActive('heading', { level: 1 })) {
          return 'format-header-1';
        }
        if (this.editor.isActive('heading', { level: 2 })) {
          return 'format-header-2';
        }
        if (this.editor.isActive('heading', { level: 3 })) {
          return 'format-header-3';
        }
      }
      return 'format-paragraph';
    },
    focusedChain() {
      return this.editor.chain().focus();
    },
  },

  watch: {
    modelValue(value) {
      const isSame = this.editor.getHTML() === value;
      if (isSame) {
        return;
      }

      this.editor.commands.setContent(value, false);
    },
  },

  mounted() {
    this.editor = new Editor({
      extensions: [StarterKit, TextAlign],
      content: this.modelValue,
      onUpdate: () => {
        // HTML
        this.$emit('update:modelValue', this.editor.getHTML());
      },
    });
  },

  beforeUnmount() {
    this.editor.destroy();
  },
};
</script>
<style lang="scss">
@import '../assets/editor.scss';
</style>
