<template>
  <div class="items">
    <oat-field :label="$t('settings.performance.concurrency')"
      :message="$t('settings.performance.concurrencyHelp')">
      <input aria-label="field" type="number" v-model.number="data['app.concurrency']" name="app.concurrency" placeholder="5" min="1"
        max="10000">
    </oat-field>

    <oat-field :label="$t('settings.performance.messageRate')"
      :message="$t('settings.performance.messageRateHelp')">
      <input aria-label="field" type="number" v-model.number="data['app.message_rate']" name="app.message_rate" placeholder="5" min="1"
        max="100000">
    </oat-field>

    <oat-field :label="$t('settings.performance.batchSize')"
      :message="$t('settings.performance.batchSizeHelp')">
      <input aria-label="field" type="number" v-model.number="data['app.batch_size']" name="app.batch_size" placeholder="1000" min="1"
        max="100000">
    </oat-field>

    <oat-field :label="$t('settings.performance.maxErrThreshold')"
      :message="$t('settings.performance.maxErrThresholdHelp')">
      <input aria-label="field" type="number" v-model.number="data['app.max_send_errors']" name="app.max_send_errors" placeholder="1999"
        min="0" max="100000">
    </oat-field>

    <div>
      <div class="row">
        <div class="col-6">
          <oat-field :message="$t('settings.performance.slidingWindowHelp')">
            <oat-switch v-model="data['app.message_sliding_window']" name="app.message_sliding_window">
              {{ $t('settings.performance.slidingWindow') }}
            </oat-switch>
          </oat-field>
        </div>

        <div class="col-3" :class="{ disabled: !data['app.message_sliding_window'] }">
          <oat-field :label="$t('settings.performance.slidingWindowRate')"
            :message="$t('settings.performance.slidingWindowRateHelp')">
            <input aria-label="field" type="number" v-model.number="data['app.message_sliding_window_rate']"
              name="sliding_window_rate" :disabled="!data['app.message_sliding_window']" placeholder="25" min="1"
              max="10000000">
          </oat-field>
        </div>

        <div class="col-3" :class="{ disabled: !data['app.message_sliding_window'] }">
          <oat-field :label="$t('settings.performance.slidingWindowDuration')"
            :message="$t('settings.performance.slidingWindowDurationHelp')">
            <input aria-label="field" v-model="data['app.message_sliding_window_duration']" name="sliding_window_duration"
              :disabled="!data['app.message_sliding_window']" placeholder="1h" :pattern="regDuration" :maxlength="10">
          </oat-field>
        </div>
      </div>
    </div><!-- sliding window -->

    <div>
      <hr />
      <div class="row">
        <div class="col-4">
          <oat-field :message="$t('settings.performance.cacheSlowQueriesHelp')">
            <oat-switch v-model="data['app.cache_slow_queries']" name="app.cache_slow_queries">
              {{ $t('settings.performance.cacheSlowQueries') }}
            </oat-switch>
          </oat-field>
        </div>
        <div class="col-4" :class="{ disabled: !data['app.cache_slow_queries'] }">
          <oat-field :label="$t('settings.maintenance.cron')">
            <input aria-label="field" v-model="data['app.cache_slow_queries_interval']" :disabled="!data['app.cache_slow_queries']"
              placeholder="0 3 * * *">
          </oat-field>
        </div>
        <div class="col-4">
          <br /><br />
          <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferer">
            <oat-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
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
