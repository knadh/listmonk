<template>
  <div v-if="active" class="b-loading" :class="{ overlay }" aria-busy="true" data-spinner="large" />
</template>

<script>
export default {
  name: 'BLoading',
  props: {
    active: {
      type: Boolean,
      default: true,
    },
    isFullPage: {
      type: Boolean,
      default: false,
    },
  },
  computed: {
    overlay() {
      return this.isFullPage;
    },
  },
};
</script>

<style>
.b-loading.overlay {
  align-items: center;
  background: color-mix(in srgb, var(--background) 70%, transparent);
  display: flex;
  inset: 0;
  justify-content: center;
  position: fixed;
  z-index: 1000;
}
</style>
