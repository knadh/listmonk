<template>
  <div data-field>
    <div class="oat-tag-input">
      <span v-for="(tag, i) in value" :key="`${tag}-${i}`" class="badge">
        {{ tag }}
        <button type="button" class="ghost small" :disabled="disabled" @click="remove(i)">x</button>
      </span>
      <input
        v-bind="$attrs"
        v-model="draft"
        type="text"
        :name="name"
        :aria-label="placeholder || name || 'Tags'"
        :placeholder="placeholder"
        :disabled="disabled"
        @keydown.enter.prevent="add"
        @keydown.tab="add"
        @keydown.backspace="onBackspace"
        @keyup="onKeyup"
      >
    </div>
  </div>
</template>

<script>
export default {
  name: 'OatTagInput',
  inheritAttrs: false,
  props: {
    value: {
      type: Array,
      default: () => [],
    },
    name: {
      type: String,
      default: '',
    },
    placeholder: {
      type: String,
      default: '',
    },
    beforeAdding: {
      type: Function,
      default: null,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  data() {
    return {
      draft: '',
    };
  },
  methods: {
    add() {
      const val = this.draft.replace(/,$/, '').trim();
      if (!val) {
        return;
      }
      if (this.beforeAdding && !this.beforeAdding(val)) {
        return;
      }
      this.$emit('input', [...this.value, val]);
      this.draft = '';
    },
    remove(i) {
      this.$emit('input', this.value.filter((_, n) => n !== i));
    },
    onBackspace() {
      if (!this.draft && this.value.length > 0) {
        this.remove(this.value.length - 1);
      }
    },
    onKeyup(e) {
      if (e.key === ',') {
        this.add();
      }
    },
  },
};
</script>
