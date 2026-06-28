<template>
  <span class="icon" :class="`mdi-${icon}`" aria-hidden="true" />
</template>

<script>
export default {
  name: 'BIcon',
  props: {
    icon: {
      type: String,
      required: true,
    },
  },
};
</script>
