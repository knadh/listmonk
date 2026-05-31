<template>
  <label>
    <input type="checkbox" :checked="checked" :disabled="disabled" @change="onChange">
    <slot />
  </label>
</template>

<script>
export default {
  name: 'OatCheckcard',
  props: {
    value: {
      type: [Array, Boolean],
      default: false,
    },
    nativeValue: {
      type: [String, Number, Boolean],
      default: true,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  computed: {
    checked() {
      return Array.isArray(this.value) ? this.value.includes(this.nativeValue) : !!this.value;
    },
  },
  methods: {
    onChange(e) {
      if (!Array.isArray(this.value)) {
        this.$emit('input', e.target.checked);
        return;
      }
      const next = this.value.filter((v) => v !== this.nativeValue);
      if (e.target.checked) {
        next.push(this.nativeValue);
      }
      this.$emit('input', next);
    },
  },
};
</script>
