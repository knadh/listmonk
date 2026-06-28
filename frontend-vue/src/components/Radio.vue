<template>
  <label>
    <input v-bind="$attrs" type="radio" :name="name" :value="nativeValue" :checked="value === nativeValue"
      :disabled="disabled" @change="$emit('input', nativeValue)">
    <slot />
  </label>
</template>

<script>
export default {
  name: 'BRadio',
  inheritAttrs: false,
  props: {
    value: {
      type: [String, Number, Boolean],
      default: null,
    },
    nativeValue: {
      type: [String, Number, Boolean],
      default: '',
    },
    name: {
      type: String,
      default: '',
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
};
</script>
