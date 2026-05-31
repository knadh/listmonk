<template>
  <div class="oat-autocomplete">
    <input v-bind="$attrs" :value="value" type="text" :placeholder="placeholder" :disabled="disabled"
      :aria-label="placeholder || 'Search'" @input="onInput" @focus="isOpen = true">
    <button v-if="clearable && value" type="button" class="ghost small" @click="clear">x</button>
    <menu v-if="isOpen && data.length > 0">
      <li v-for="(item, i) in data" :key="i">
        <button type="button" class="ghost" @click="select(item)">
          <slot name="default" :option="item">
            {{ item[field] || item }}
          </slot>
        </button>
      </li>
    </menu>
  </div>
</template>

<script>
export default {
  name: 'OatAutocomplete',
  inheritAttrs: false,
  props: {
    value: {
      type: String,
      default: '',
    },
    data: {
      type: Array,
      default: () => [],
    },
    field: {
      type: String,
      default: 'name',
    },
    placeholder: {
      type: String,
      default: '',
    },
    clearable: {
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
      isOpen: false,
    };
  },
  methods: {
    onInput(e) {
      this.isOpen = true;
      this.$emit('input', e.target.value);
      this.$emit('typing', e.target.value);
    },
    select(item) {
      this.isOpen = false;
      this.$emit('select', item);
      this.$emit('input', item[this.field] || item);
    },
    clear() {
      this.$emit('input', '');
      this.$emit('select', null);
    },
  },
};
</script>

<style>
.oat-autocomplete {
  position: relative;
}

.oat-autocomplete menu {
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

.oat-autocomplete menu button {
  justify-content: start;
  width: 100%;
}
</style>
