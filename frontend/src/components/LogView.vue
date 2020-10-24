<template>
    <section class="log-view">
    <b-loading :active="loading" :is-full-page="false" />
    <pre class="lines" ref="lines">
<template v-for="(l, i) in lines"><span v-html="formatLine(l)" :key="i" class="line"></span>
</template></pre>
    </section>
</template>


<script>
const reFormatLine = new RegExp(/^(.*) (.+?)\.go:[0-9]+:\s/g);

export default {
  name: 'LogView',

  props: {
    loading: Boolean,
    lines: {
      type: Array,
      default: () => [],
    },
  },

  methods: {
    formatLine: (l) => l.replace(reFormatLine, '<span class="stamp">$1</span> '),
  },

  watch: {
    lines() {
      this.$nextTick(() => {
        this.$refs.lines.scrollTop = this.$refs.lines.scrollHeight;
      });
    },
  },
};
</script>
