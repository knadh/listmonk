<template>
  <div data-field>
    <span v-if="label">{{ label }}</span>
    <slot />
    <small v-if="message" data-hint>{{ message }}</small>
  </div>
</template>

<script>
export default {
  name: 'OatField',
  props: {
    label: {
      type: String,
      default: '',
    },
    message: {
      type: String,
      default: '',
    },
  },
};
</script>
