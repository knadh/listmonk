<template>
  <section class="maintenance wrap">
    <h1 class="title is-4">
      {{ $t("maintenance.title") }}
    </h1>
    <hr />
    <p class="has-text-grey">
      {{ $t("maintenance.help") }}
    </p>
    <br />

    <div class="box">
      <h4 class="is-size-4">
        {{ $t("globals.terms.subscribers") }}
      </h4>
      <br />
      <div class="columns">
        <div class="column is-4">
          <b-field label="Data" :message="$t('maintenance.orphanHelp')">
            <b-select v-model="subscriberType" expanded>
              <option value="orphan">
                {{ $t("dashboard.orphanSubs") }}
              </option>
              <option value="blocklisted">
                {{ $t("subscribers.status.blocklisted") }}
              </option>
            </b-select>
          </b-field>
        </div>
        <div class="column is-5" />
        <div class="column">
          <br />
          <b-field>
            <b-button
              class="is-primary"
              :loading="loading.maintenance"
              @click="deleteSubscribers"
              expanded
            >
              {{ $t("globals.buttons.delete") }}
            </b-button>
          </b-field>
          <hr />
          <b-field>
            <b-button
              class="is-primary"
              :loading="loading.maintenance"
              @click="exportSubscribers"
              expanded
            >
              Export All Subscribers
            </b-button>
          </b-field>
        </div>
      </div>
    </div>
    <!-- subscribers -->

    <div class="box mt-6">
      <h4 class="is-size-4">
        {{ $tc("globals.terms.subscriptions", 2) }}
      </h4>
      <br />
      <div class="columns">
        <div class="column is-4">
          <b-field label="Data">
            <b-select v-model="subscriptionType" expanded>
              <option value="optin">
                {{ $t("maintenance.maintenance.unconfirmedOptins") }}
              </option>
            </b-select>
          </b-field>
        </div>
        <div class="column is-4">
          <b-field :label="$t('maintenance.olderThan')">
            <b-datepicker
              v-model="subscriptionDate"
              required
              expanded
              icon="calendar-clock"
              :date-formatter="formatDateTime"
            />
          </b-field>
        </div>
        <div class="column is-1" />
        <div class="column">
          <br />
          <b-field>
            <b-button
              class="is-primary"
              :loading="loading.maintenance"
              @click="deleteSubscriptions"
              expanded
            >
              {{ $t("globals.buttons.delete") }}
            </b-button>
          </b-field>
        </div>
      </div>
    </div>
    <!-- subscriptions -->

    <div class="box mt-6">
      <h4 class="is-size-4">
        {{ $t("globals.terms.analytics") }}
      </h4>
      <br />
      <div class="columns">
        <div class="column is-4">
          <b-field label="Data">
            <b-select v-model="analyticsType" expanded>
              <option selected value="all">
                {{ $t("globals.terms.all") }}
              </option>
              <option value="views">
                {{ $t("dashboard.campaignViews") }}
              </option>
              <option value="clicks">
                {{ $t("dashboard.linkClicks") }}
              </option>
            </b-select>
          </b-field>
        </div>
        <div class="column is-4">
          <b-field :label="$t('maintenance.olderThan')">
            <b-datepicker
              v-model="analyticsDate"
              required
              expanded
              icon="calendar-clock"
              :date-formatter="formatDateTime"
            />
          </b-field>
        </div>
        <div class="column is-1" />
        <div class="column">
          <br />
          <b-field>
            <b-button
              expanded
              class="is-primary"
              :loading="loading.maintenance"
              @click="deleteAnalytics"
            >
              {{ $t("globals.buttons.delete") }}
            </b-button>
          </b-field>
        </div>
      </div>

      <div class="columns">
        <div class="column is-4">
          <b-field label="Data">
            <b-select v-model="analyticsTypeExport" expanded>
              <option selected value="all">
                {{ $t("globals.terms.all") }}
              </option>
              <option value="views">
                {{ $t("dashboard.campaignViews") }}
              </option>
              <option value="clicks">
                {{ $t("dashboard.linkClicks") }}
              </option>
            </b-select>
          </b-field>
        </div>
        <div class="column is-4">
          <b-field label="From">
            <b-datepicker
              v-model="analyticsDateFrom"
              required
              expanded
              icon="calendar-clock"
              :date-formatter="formatDateTime"
            />
          </b-field>
          <b-field label="To">
            <b-datepicker
              v-model="analyticsDateTo"
              required
              expanded
              icon="calendar-clock"
              :date-formatter="formatDateTime"
            />
          </b-field>
        </div>
        <div class="column is-1" />
        <div class="column">
          <br />
          <b-field>
            <b-button
              expanded
              class="is-primary"
              :loading="loading.maintenance"
              @click="exportAnalytics"
            >
              {{ $t("globals.buttons.export") }}
            </b-button>
          </b-field>
        </div>
      </div>
    </div>
    <!-- analytics -->
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  components: {},

  data() {
    return {
      subscriberType: 'orphan',
      analyticsType: 'all',
      analyticsTypeExport: 'all',
      subscriptionType: 'optin',
      analyticsDate: dayjs().subtract(7, 'day').toDate(),
      analyticsDateFrom: dayjs().subtract(14, 'day').toDate(),
      analyticsDateTo: dayjs().toDate(),
      campaignId: 1,
      subscriptionDate: dayjs().subtract(7, 'day').toDate(),
    };
  },

  methods: {
    formatDateTime(s) {
      return dayjs(s).format('YYYY-MM-DD');
    },

    deleteSubscribers() {
      this.$utils.confirm(null, () => {
        this.$api.deleteGCSubscribers(this.subscriberType).then((data) => {
          this.$utils.toast(
            this.$t('globals.messages.deletedCount', {
              name: this.$tc('globals.terms.subscribers', 2),
              num: data.count,
            }),
          );
        });
      });
    },

    exportSubscribers() {
      this.$api.getSubscribers().then((data) => {
        const subscribersData = data.results;
        if (subscribersData.length === 0) {
          this.$utils.toast('No Subscribers available!', 'error');
        } else {
          this.$utils.downloadCSV(subscribersData);
          this.$utils.toast('Successfully exported subscribers data');
        }
      });
    },

    deleteSubscriptions() {
      this.$utils.confirm(null, () => {
        this.$api.deleteGCSubscriptions(this.subscriptionDate).then((data) => {
          this.$utils.toast(
            this.$t('globals.messages.deletedCount', {
              name: this.$tc('globals.terms.subscriptions', 2),
              num: data.count,
            }),
          );
        });
      });
    },

    deleteAnalytics() {
      this.$utils.confirm(null, () => {
        this.$api
          .deleteGCCampaignAnalytics(this.analyticsType, this.analyticsDate)
          .then(() => {
            this.$utils.toast(this.$t('globals.messages.done'));
          });
      });
    },

    exportAnalytics() {
      if (this.analyticsDateTo < this.analyticsDateFrom) {
        this.$utils.toast("'From' Date should be less than 'To' Date", 'error');
        return;
      }
      if (
        this.analyticsTypeExport === 'views'
        || this.analyticsTypeExport === 'all'
      ) {
        this.$api
          .getGCCampaignAnalyticsViews(
            1,
            this.analyticsDateFrom,
            this.analyticsDateTo,
          )
          .then((data) => {
            if (data.length === 0) {
              this.$utils.toast('No analytics found!', 'error');
              return;
            }

            this.$utils.downloadCSV(data, 'campaign_views');
            this.$utils.toast('Successfully exported campaign views');
          })
          .catch((err) => {
            this.$utils.toast(err, 'error');
          });
      }

      if (
        this.analyticsTypeExport === 'clicks'
        || this.analyticsTypeExport === 'all'
      ) {
        this.$api
          .getGCCampaignAnalyticsLinkClicks(
            1,
            this.analyticsDateFrom,
            this.analyticsDateTo,
          )
          .then((data) => {
            if (data.length === 0) {
              this.$utils.toast('No analytics found!', 'error');
              return;
            }

            this.$utils.downloadCSV(data, 'link_clicks');
            this.$utils.toast('Successfully exported link clicks data');
          })
          .catch((err) => {
            this.$utils.toast(err, 'error');
          });
      }
    },
  },

  computed: {
    ...mapState(['loading']),
  },
});
</script>
