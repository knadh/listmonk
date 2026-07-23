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
          <b-field :message="$t('settings.performance.slidingWindowHelp')">
            <b-switch v-model="data['app.message_sliding_window']" name="app.message_sliding_window">
              {{ $t('settings.performance.slidingWindow') }}
            </b-switch>
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

    <div>
      <hr />
      <div class="columns">
        <div class="column is-12">
          <b-field :message="$t('settings.performance.sendCalendarHelp')">
            <b-switch v-model="data['app.message_send_calendar']" name="app.message_send_calendar">
              {{ $t('settings.performance.sendCalendar') }}
            </b-switch>
          </b-field>
        </div>
      </div>

      <div v-if="data['app.message_send_calendar']" style="margin-top: 15px; margin-bottom: 20px;">
        <table class="table is-narrow is-fullwidth" style="background: transparent;">
          <thead>
            <tr>
              <th style="width: 30%">Day</th>
              <th style="width: 35%">{{ $t('settings.performance.startAt') }}</th>
              <th style="width: 35%">{{ $t('settings.performance.endAt') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="day in weekdays" :key="day">
              <td style="vertical-align: middle;">
                <b-switch v-model="data['app.message_calendar_schedule'][day].enabled">
                  <span class="is-capitalized">{{ day }}</span>
                </b-switch>
              </td>
              <td>
                <b-field>
                  <b-input v-model="data['app.message_calendar_schedule'][day].start_at" type="time"
                    :disabled="!data['app.message_calendar_schedule'][day].enabled" size="is-small" />
                </b-field>
              </td>
              <td>
                <b-field>
                  <b-input v-model="data['app.message_calendar_schedule'][day].end_at" type="time"
                    :disabled="!data['app.message_calendar_schedule'][day].enabled" size="is-small" />
                </b-field>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div><!-- global sending calendar -->

    <div>
      <hr />
      <div class="columns">
        <div class="column is-4">
          <b-field :message="$t('settings.performance.cacheSlowQueriesHelp')">
            <b-switch v-model="data['app.cache_slow_queries']" name="app.cache_slow_queries">
              {{ $t('settings.performance.cacheSlowQueries') }}
            </b-switch>
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
      weekdays: ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'],
    };
  },

  created() {
    if (!this.data['app.message_calendar_schedule']) {
      Vue.set(this.data, 'app.message_calendar_schedule', {});
    }
    this.weekdays.forEach((day) => {
      if (!this.data['app.message_calendar_schedule'][day]) {
        Vue.set(this.data['app.message_calendar_schedule'], day, {
          enabled: false,
          start_at: '09:00',
          end_at: '17:00',
        });
      }
    });
  },
});
</script>
