<template>
  <section class="app-section">
    <div class="content text-light align-center">
      <p>
        <oat-icon :icon="!icon ? 'plus' : icon" />
      </p>
      <p>{{ !label ? $t('globals.messages.emptyState') : label }}</p>
    </div>
  </section>
</template>

<script>
export default {
  name: 'EmptyPlaceholder',

  props: {
    icon: { type: String, default: '' },
    label: { type: String, default: '' },
  },
};
</script>
