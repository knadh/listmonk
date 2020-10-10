<template>
  <section class="logs content relative">
    <h1 class="title is-4">Logs</h1>
    <hr />
    <b-loading :active="loading.logs" :is-full-page="false" />
    <pre class="lines" ref="lines">
<template v-for="(l, i) in lines"><span v-html="formatLine(l)" :key="i" class="line"></span>
</template>
    </pre>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

const reFormatLine = new RegExp(/^(.*) (.+?)\.go:[0-9]+:\s/g);

export default Vue.extend({
  data() {
    return {
      lines: '',
      pollId: null,
    };
  },

  methods: {
    formatLine: (l) => l.replace(reFormatLine, '<span class="stamp">$1</span> '),

    getLogs() {
      this.$api.getLogs().then((data) => {
        this.lines = data;

        this.$nextTick(() => {
          this.$refs.lines.scrollTop = this.$refs.lines.scrollHeight;
        });
      });
    },
  },

  computed: {
    ...mapState(['logs', 'loading']),
  },

  mounted() {
    this.getLogs();

    // Update the logs every 10 seconds.
    this.pollId = setInterval(() => this.getLogs(), 10000);
  },

  destroyed() {
    clearInterval(this.pollId);
  },
});
</script>
