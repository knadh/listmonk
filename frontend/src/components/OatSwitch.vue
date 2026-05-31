<template>
  <label>
    <input
      v-bind="$attrs"
      type="checkbox"
      role="switch"
      :checked="isChecked"
      :aria-checked="isChecked ? 'true' : 'false'"
      :disabled="disabled"
      @change="onChange"
    >
    <slot />
  </label>
</template>

<script>
export default {
  name: 'OatSwitch',
  inheritAttrs: false,
  props: {
    value: {
      type: [Boolean, String, Number],
      default: false,
    },
    nativeValue: {
      type: [Boolean, String, Number],
      default: true,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  computed: {
    isChecked() {
      return this.value === this.nativeValue || this.value === true;
    },
  },
  methods: {
    onChange(e) {
      this.$emit('input', e.target.checked ? this.nativeValue : false);
      this.$emit('change', e.target.checked);
    },
  },
};
</script>
