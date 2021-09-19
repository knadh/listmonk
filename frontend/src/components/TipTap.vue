<template>
  <div class="tiptap-editor">
    <div v-if="editor" class="editor-menu">
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleBold()
            .run()
        "
        :class="{ 'is-active': editor.isActive('bold') }"
      >
        bold
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleItalic()
            .run()
        "
        :class="{ 'is-active': editor.isActive('italic') }"
      >
        italic
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleStrike()
            .run()
        "
        :class="{ 'is-active': editor.isActive('strike') }"
      >
        strike
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleCode()
            .run()
        "
        :class="{ 'is-active': editor.isActive('code') }"
      >
        code
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .setParagraph()
            .run()
        "
        :class="{ 'is-active': editor.isActive('paragraph') }"
      >
        paragraph
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleHeading({ level: 1 })
            .run()
        "
        :class="{ 'is-active': editor.isActive('heading', { level: 1 }) }"
      >
        h1
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleHeading({ level: 2 })
            .run()
        "
        :class="{ 'is-active': editor.isActive('heading', { level: 2 }) }"
      >
        h2
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleHeading({ level: 3 })
            .run()
        "
        :class="{ 'is-active': editor.isActive('heading', { level: 3 }) }"
      >
        h3
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleBulletList()
            .run()
        "
        :class="{ 'is-active': editor.isActive('bulletList') }"
      >
        bullet list
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleOrderedList()
            .run()
        "
        :class="{ 'is-active': editor.isActive('orderedList') }"
      >
        ordered list
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleCodeBlock()
            .run()
        "
        :class="{ 'is-active': editor.isActive('codeBlock') }"
      >
        code block
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .toggleBlockquote()
            .run()
        "
        :class="{ 'is-active': editor.isActive('blockquote') }"
      >
        blockquote
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .setHorizontalRule()
            .run()
        "
      >
        horizontal rule
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .setHardBreak()
            .run()
        "
      >
        hard break
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .undo()
            .run()
        "
      >
        undo
      </b-button>
      <b-button
        size="is-small"
        @click="
          editor
            .chain()
            .focus()
            .redo()
            .run()
        "
      >
        redo
      </b-button>
    </div>
    <editor-content :editor="editor" class="tiptap-editor__content p-2" />
  </div>
</template>

<script>
import { Editor, EditorContent } from '@tiptap/vue-2';
import StarterKit from '@tiptap/starter-kit';

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
    };
  },

  watch: {
    modelValue(value) {
      // HTML
      const isSame = this.editor.getHTML() === value;

      // JSON
      // const isSame = this.editor.getJSON().toString() === value.toString()

      if (isSame) {
        return;
      }

      this.editor.commands.setContent(value, false);
    },
  },

  mounted() {
    this.editor = new Editor({
      extensions: [StarterKit],
      content: this.modelValue,
      onUpdate: () => {
        // HTML
        this.$emit('update:modelValue', this.editor.getHTML());

        // JSON
        // this.$emit('update:modelValue', this.editor.getJSON())
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
