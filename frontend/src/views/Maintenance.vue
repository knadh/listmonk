<template>
  <section class="maintenance">
    <header class="row page-header">
      <div class="col-8">
        <h1>
          {{ $t('maintenance.title') }}
        </h1>
        <p class="text-light text-7">
          {{ $t('maintenance.help') }}
        </p>
      </div>
    </header>

    <div class="card page-content">
      <div class="card">
        <h4 class="text-4">
          {{ $t('globals.terms.subscribers') }}
        </h4><br />
        <div class="row">
          <div class="col-4">
            <oat-field label="Data" :message="$t('maintenance.orphanHelp')">
              <select aria-label="field" v-model="subscriberType">
                <option value="orphan">
                  {{ $t('dashboard.orphanSubs') }}
                </option>
                <option value="blocklisted">
                  {{ $t('subscribers.status.blocklisted') }}
                </option>
              </select>
            </oat-field>
          </div>
          <div class="col-5" />
          <div class="col-12">
            <br />
            <oat-field>
              <button type="button" data-variant="danger" :loading="loading.maintenance" @click="deleteSubscribers">
                {{ $t('globals.buttons.delete') }}
              </button>
            </oat-field>
          </div>
        </div>
      </div><!-- subscribers -->

      <div class="card mt-6">
        <h4 class="text-4">
          {{ $tc('globals.terms.subscriptions', 2) }}
        </h4><br />
        <div class="row">
          <div class="col-4">
            <oat-field label="Data">
              <select aria-label="field" v-model="subscriptionType">
                <option value="optin">
                  {{ $t('maintenance.maintenance.unconfirmedOptins') }}
                </option>
              </select>
            </oat-field>
          </div>
          <div class="col-4">
            <oat-field :label="$t('maintenance.olderThan')">
              <oat-date-input v-model="subscriptionDate" required />
            </oat-field>
          </div>
          <div class="col-1" />
          <div class="col-12">
            <br />
            <oat-field>
              <button type="button" data-variant="danger" :loading="loading.maintenance" @click="deleteSubscriptions">
                {{ $t('globals.buttons.delete') }}
              </button>
            </oat-field>
          </div>
        </div>
      </div><!-- subscriptions -->

      <div class="card mt-6">
        <h4 class="text-4">
          {{ $t('globals.terms.analytics') }}
        </h4><br />
        <div class="row">
          <div class="col-4">
            <oat-field label="Data">
              <select aria-label="field" v-model="analyticsType">
                <option selected value="all">
                  {{ $t('globals.terms.all') }}
                </option>
                <option value="views">
                  {{ $t('dashboard.campaignViews') }}
                </option>
                <option value="clicks">
                  {{ $t('dashboard.linkClicks') }}
                </option>
              </select>
            </oat-field>
          </div>
          <div class="col-4">
            <oat-field :label="$t('maintenance.olderThan')">
              <oat-date-input v-model="analyticsDate" required />
            </oat-field>
          </div>
          <div class="col-1" />
          <div class="col-12">
            <br />
            <oat-field>
              <button type="button" data-variant="danger" :loading="loading.maintenance" @click="deleteAnalytics">
                {{ $t('globals.buttons.delete') }}
              </button>
            </oat-field>
          </div>
        </div>

        <hr />
        <h5 class="text-5">
          {{ $t('subscribers.export') }}
        </h5>
        <br />
        <div class="row">
          <div class="col-4">
            <oat-field label="Data">
              <select aria-label="field" v-model="exportType">
                <option value="views">
                  {{ $t('dashboard.campaignViews') }}
                </option>
                <option value="clicks">
                  {{ $t('dashboard.linkClicks') }}
                </option>
              </select>
            </oat-field>
          </div>
          <div class="col-4">
            <oat-field :label="$t('analytics.fromDate')">
              <oat-date-input v-model="exportDate" required />
            </oat-field>
          </div>
          <div class="col-1" />
          <div class="col-12">
            <br />
            <oat-field>
              <button type="button" data-variant="primary" tag="a" :href="exportURL">
                {{ $t('subscribers.export') }}
              </button>
            </oat-field>
          </div>
        </div>
      </div><!-- analytics -->

      <form @submit.prevent="onUpdateDBSettings" class="card mt-6">
        <h4 class="text-4">
          {{ $t('maintenance.database.title') }}
        </h4><br />
        <h5 class="text-5">Vacuum</h5>
        <p class="text-light text-7 ">
          {{ $t('maintenance.database.vacuumHelp') }}
        </p>
        <br />
        <div class="row">
          <div class="col-2">
            <oat-field :label="$t('globals.buttons.enabled')">
              <oat-switch v-model="dbSettings.vacuum" />
            </oat-field>
          </div>
          <div class="col-4" :class="{ disabled: !dbSettings.vacuum }">
            <oat-field :label="$t('settings.maintenance.cron')">
              <input aria-label="field" v-model="dbSettings.vacuum_cron_interval" placeholder="0 2 * * *"
                :disabled="!dbSettings.vacuum" pattern="((\*|[0-9,\-\/]+)\s+){4}(\*|[0-9,\-\/]+)">
            </oat-field>
          </div>
          <div class="col-3" />
          <div class="col-3">
            <br />
            <button data-variant="primary" type="submit" :loading="loading.settings">
              {{ $t('globals.buttons.save') }}
            </button>
          </div>
        </div>
      </form><!-- database -->

      <oat-loading :is-full-page="true" v-if="isLoading" active />
    </div>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  components: {
  },

  data() {
    return {
      isLoading: false,
      subscriberType: 'orphan',
      analyticsType: 'all',
      subscriptionType: 'optin',
      analyticsDate: dayjs().subtract(7, 'day').toDate(),
      subscriptionDate: dayjs().subtract(7, 'day').toDate(),
      exportType: 'views',
      exportDate: dayjs().subtract(30, 'day').toDate(),
      dbSettings: {
        vacuum: false,
        vacuum_cron_interval: '0 2 * * *',
      },
    };
  },

  mounted() {
    this.loadDBSettings();
  },

  methods: {
    formatDateTime(s) {
      return dayjs(s).format('YYYY-MM-DD');
    },

    deleteSubscribers() {
      this.$utils.confirm(
        null,
        () => {
          this.$api.deleteGCSubscribers(this.subscriberType).then((data) => {
            this.$utils.toast(this.$t(
              'globals.messages.deletedCount',
              { name: this.$tc('globals.terms.subscribers', 2), num: data.count },
            ));
          });
        },
      );
    },

    deleteSubscriptions() {
      this.$utils.confirm(
        null,
        () => {
          this.$api.deleteGCSubscriptions(this.subscriptionDate).then((data) => {
            this.$utils.toast(this.$t(
              'globals.messages.deletedCount',
              { name: this.$tc('globals.terms.subscriptions', 2), num: data.count },
            ));
          });
        },
      );
    },

    deleteAnalytics() {
      this.$utils.confirm(
        null,
        () => {
          this.$api.deleteGCCampaignAnalytics(this.analyticsType, this.analyticsDate)
            .then(() => {
              this.$utils.toast(this.$t('globals.messages.done'));
            });
        },
      );
    },

    loadDBSettings() {
      this.$api.getSettings().then((data) => {
        if (data['maintenance.db'] !== undefined) {
          this.dbSettings = { ...data['maintenance.db'] };
        }
      });
    },

    async onUpdateDBSettings() {
      this.isLoading = true;
      const data = await this.$api.updateSettingsByKey('maintenance.db', this.dbSettings);
      await this.$root.awaitRestart(data);
      this.isLoading = false;
    },
  },

  computed: {
    ...mapState(['loading']),

    exportURL() {
      const since = encodeURIComponent(dayjs(this.exportDate).toISOString());
      return `/api/maintenance/analytics/${this.exportType}/export?since=${since}`;
    },
  },

});
</script>
