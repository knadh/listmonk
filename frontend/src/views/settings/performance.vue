<template>
  <div class="items">
    <b-field :label="$t('settings.performance.concurrency')" label-position="on-border"
      :message="$t('settings.performance.concurrencyHelp')">
      <b-numberinput v-model="data['app.concurrency']" name="app.concurrency" type="is-light" placeholder="5" min="1"
        max="10000" />
    </b-field>

    <b-field :label="$t('settings.performance.messageRate')" label-position="on-border"
      :message="$t('settings.performance.messageRateHelp')">
      <b-numberinput v-model="data['app.message_rate']" name="app.message_rate" type="is-light" placeholder="5" min="1"
        max="100000" />
    </b-field>

    <b-field :label="$t('settings.performance.batchSize')" label-position="on-border"
      :message="$t('settings.performance.batchSizeHelp')">
      <b-numberinput v-model="data['app.batch_size']" name="app.batch_size" type="is-light" placeholder="1000" min="1"
        max="100000" />
    </b-field>

    <b-field :label="$t('settings.performance.maxErrThreshold')" label-position="on-border"
      :message="$t('settings.performance.maxErrThresholdHelp')">
      <b-numberinput v-model="data['app.max_send_errors']" name="app.max_send_errors" type="is-light" placeholder="1999"
        min="0" max="100000" />
    </b-field>

    <div>
      <hr />
      <div class="columns">
        <div class="column is-4">
          <b-field :label="$t('settings.performance.cacheSlowQueries')"
            :message="$t('settings.performance.cacheSlowQueriesHelp')">
            <b-switch v-model="data['app.cache_slow_queries']" name="app.cache_slow_queries" />
          </b-field>
        </div>
        <div class="column is-4" :class="{ disabled: !data['app.cache_slow_queries'] }">
          <b-field :label="$t('settings.maintenance.cron')">
            <b-input v-model="data['app.cache_slow_queries_interval']" :disabled="!data['app.cache_slow_queries']"
              placeholder="0 3 * * *" />
          </b-field>
        </div>
        <div class="column">
          <br /><br />
          <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferer">
            <b-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';
import { regDuration } from '../../constants';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
      regDuration,
    };
  },
});
</script>
