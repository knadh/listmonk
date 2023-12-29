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
      <div class="columns">
        <div class="column is-6">
          <b-field :label="$t('settings.performance.slidingWindow')"
            :message="$t('settings.performance.slidingWindowHelp')">
            <b-switch v-model="data['app.message_sliding_window']" name="app.message_sliding_window" />
          </b-field>
        </div>

        <div class="column is-3" :class="{ disabled: !data['app.message_sliding_window'] }">
          <b-field :label="$t('settings.performance.slidingWindowRate')" label-position="on-border"
            :message="$t('settings.performance.slidingWindowRateHelp')">
            <b-numberinput v-model="data['app.message_sliding_window_rate']" name="sliding_window_rate" type="is-light"
              controls-position="compact" :disabled="!data['app.message_sliding_window']" placeholder="25" min="1"
              max="10000000" />
          </b-field>
        </div>

        <div class="column is-3" :class="{ disabled: !data['app.message_sliding_window'] }">
          <b-field :label="$t('settings.performance.slidingWindowDuration')" label-position="on-border"
            :message="$t('settings.performance.slidingWindowDurationHelp')">
            <b-input v-model="data['app.message_sliding_window_duration']" name="sliding_window_duration"
              :disabled="!data['app.message_sliding_window']" placeholder="1h" :pattern="regDuration" :maxlength="10" />
          </b-field>
        </div>
      </div>
    </div><!-- sliding window -->
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
