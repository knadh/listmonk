<template>
  <span class="icon" :class="`mdi-${icon}`" aria-hidden="true" />
</template>

<script>
export default {
  name: 'OatIcon',
  props: {
    icon: {
      type: String,
      required: true,
    },
  },
};
</script>
