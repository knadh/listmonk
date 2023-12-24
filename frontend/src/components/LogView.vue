<template>
  <section class="log-view">
    <b-loading :active="loading" :is-full-page="false" />
    <div class="lines" ref="lines">
      <template v-for="(l, i) in lines">
        <span :set="line = splitLine(l)" :key="i" class="line">
          <span class="timestamp" :title="line.file">{{ line.timestamp }}</span>
          <span class="log-message">{{ line.message }}</span>
        </span>
      </template>
    </div>
  </section>
</template>

<script>
// Regexp for splitting log lines in the following format to
// [timestamp] [file] [message].
// 2021/05/01 00:00:00 init.go:99: reading config: config.toml
const reFormatLine = /^([0-9\s:/]+) (.+?\.go:[0-9]+):\s/g;

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
    splitLine: (l) => {
      const parts = l.split(reFormatLine);
      return {
        timestamp: parts[1],
        file: parts[2],
        message: parts[3],
      };
    },

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
