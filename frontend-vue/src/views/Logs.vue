<template>
  <section class="logs relative">
    <header class="row page-header">
      <div class="col-8">
        <h1>
          {{ $t('logs.title') }}
        </h1>
      </div>
    </header>

    <div class="card page-content">
      <log-view :loading="loading.logs" :lines="lines" />
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import LogView from '../components/LogView.vue';

export default Vue.extend({
  components: {
    LogView,
  },

  data() {
    return {
      lines: [],
      pollId: null,
    };
  },

  methods: {
    getLogs() {
      this.$api.getLogs().then((data) => {
        this.lines = data;
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
