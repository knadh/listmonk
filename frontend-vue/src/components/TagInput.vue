<template>
  <div data-field>
    <div class="b-taginput">
      <span v-for="(tag, i) in value" :key="`${tagLabel(tag)}-${i}`" class="badge">
        {{ tagLabel(tag) }}
        <button type="button" class="ghost small" :disabled="disabled" @click="remove(i)">x</button>
      </span>
      <input ref="input" v-bind="$attrs" v-model="draft" type="text" :name="name"
        :aria-label="placeholder || name || 'Tags'" :placeholder="placeholder" :disabled="disabled"
        @focus="onFocus" @input="$emit('typing', draft)" @keydown.enter.prevent="add"
        @keydown.tab="add" @keydown.backspace="onBackspace" @keyup="onKeyup">
      <menu v-if="autocomplete && isOpen && data.length > 0">
        <li v-for="(item, i) in data" :key="`${tagLabel(item)}-${i}`">
          <button type="button" class="ghost" @click="addItem(item)">
            {{ tagLabel(item) }}
          </button>
        </li>
      </menu>
    </div>
  </div>
</template>

<script>
export default {
  name: 'BTaginput',
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
    data: {
      type: Array,
      default: () => [],
    },
    field: {
      type: String,
      default: '',
    },
    autocomplete: {
      type: Boolean,
      default: false,
    },
    allowNew: {
      type: Boolean,
      default: true,
    },
    openOnFocus: {
      type: Boolean,
      default: false,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  data() {
    return {
      draft: '',
      isOpen: false,
    };
  },
  methods: {
    tagLabel(tag) {
      return this.field && tag && typeof tag === 'object' ? tag[this.field] : tag;
    },
    add() {
      const val = this.draft.replace(/,$/, '').trim();
      if (!val) {
        return;
      }

      if (!this.allowNew) {
        const item = this.data.find((d) => this.tagLabel(d) === val);
        if (item) {
          this.addItem(item);
        }
        return;
      }

      this.addItem(val);
    },
    addItem(item) {
      if (this.beforeAdding && !this.beforeAdding(item)) {
        return;
      }
      this.$emit('input', [...this.value, item]);
      this.draft = '';
      this.isOpen = false;
    },
    remove(i) {
      this.$emit('input', this.value.filter((_, n) => n !== i));
    },
    focus() {
      this.$refs.input.focus();
    },
    onFocus(e) {
      this.isOpen = this.openOnFocus || this.autocomplete;
      this.$emit('focus', e);
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

<style>
.b-taginput {
  position: relative;
}

.b-taginput menu {
  background: var(--background);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  left: 0;
  list-style: none;
  margin: var(--space-1) 0 0;
  max-height: 260px;
  overflow: auto;
  padding: var(--space-1);
  position: absolute;
  right: 0;
  z-index: 20;
}

.b-taginput menu button {
  justify-content: start;
  width: 100%;
}
</style>
