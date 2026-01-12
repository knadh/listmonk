<template>
  <a href="#" class="a btn-reload" @click.prevent="$emit('reload')" :data-cy="dataCy"
    :aria-label="$t('globals.buttons.reload')">
    <b-tooltip :label="$t('globals.buttons.reload')" type="is-dark">
      <b-icon icon="refresh" size="is-small" :class="{ 'spin': loading }" />
    </b-tooltip>
  </a>
</template>

<script>
export default {
  name: 'ReloadButton',

  props: {
    loading: { type: Boolean, default: false },
    dataCy: { type: String, default: 'btn-reload' },
  },
};
</script>
