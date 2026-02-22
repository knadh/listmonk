<template>
  <section class="analytics content relative">
    <h1 class="title is-4">
      {{ $t('analytics.title') }}
    </h1>
    <div v-if="serverConfig.privacy.disable_tracking || !serverConfig.privacy.individual_tracking"
      class="notification is-info">
      <template v-if="serverConfig.privacy.disable_tracking">
        {{ $t('analytics.trackingDisabled') }}
      </template>
      <template v-else-if="!serverConfig.privacy.individual_tracking">
        {{ $t('analytics.nonIndividualTracking') }}
      </template>
    </div>
    <hr />

    <form @submit.prevent="onSubmit">
      <div class="columns">
        <div class="column is-6">
          <b-field :label="$t('globals.terms.campaigns')" label-position="on-border">
            <b-taginput v-model="form.campaigns" :data="queriedCampaigns" name="campaigns" ellipsis icon="tag-outline"
              :placeholder="$t('globals.terms.campaigns')" autocomplete :allow-new="false"
              :before-adding="isCampaignSelected" @typing="queryCampaigns" field="name" :loading="isSearchLoading" />
          </b-field>
        </div>

        <div class="column is-5">
          <div class="columns">
            <div class="column is-6">
              <b-field data-cy="from" :label="$t('analytics.fromDate')" label-position="on-border">
                <b-datetimepicker v-model="form.from" icon="calendar-clock" :timepicker="{ hourFormat: '24' }"
                  :datetime-formatter="formatDateTime" @input="onFromDateChange" />
              </b-field>
            </div>
            <div class="column is-6">
              <b-field data-cy="to" :label="$t('analytics.toDate')" label-position="on-border">
                <b-datetimepicker v-model="form.to" icon="calendar-clock" :timepicker="{ hourFormat: '24' }"
                  :datetime-formatter="formatDateTime" @input="onToDateChange" />
              </b-field>
            </div>
          </div><!-- columns -->
        </div><!-- columns -->

        <div class="column is-1">
          <b-button native-type="submit" type="is-primary" icon-left="magnify" :disabled="form.campaigns.length === 0"
            data-cy="btn-search" />
        </div>
      </div><!-- columns -->
    </form>

    <section class="charts mt-5">
      <div class="chart" v-for="(v, k) in charts" :key="k">
        <div class="columns">
          <div class="column is-9">
            <b-loading v-if="v.loading" :active="v.loading" :is-full-page="false" />
            <h4 v-if="v.chart !== null">
              {{ v.name }}
              <span class="has-text-grey-light">({{ $utils.niceNumber(counts[k]) }})</span>
            </h4>
            <chart :type="v.type" v-if="!v.loading" :data="v.data" :on-click="v.onClick" />
          </div>
          <div class="column is-2 donut-container">
            <chart type="donut" v-if="!v.loading" :data="v.donutData" />
          </div>
        </div>
      </div>
    </section>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';
import { colors } from '../constants';
import Chart from '../components/Chart.vue';

const chartColorRed = '#ee7d5b';
const chartColors = [
  colors.primary,
  '#FFB50D',
  '#41AC9C',
  chartColorRed,
  '#7FC7BC',
  '#3a82d6',
  '#688ED9',
  '#FFC43D',
];

export default Vue.extend({
  components: {
    Chart,
  },

  data() {
    return {
      isSearchLoading: false,
      queriedCampaigns: [],

      // Data for each view.
      counts: {
        views: 0,
        clicks: 0,
        bounces: 0,
        links: 0,
      },
      urls: [],
      charts: {
        views: {
          name: this.$t('campaigns.views'),
          type: 'line',
          data: null,
          fn: this.$api.getCampaignViewCounts,
          chartFn: this.makeCharts,
          loading: false,
        },

        clicks: {
          name: this.$t('campaigns.clicks'),
          type: 'line',
          data: null,
          fn: this.$api.getCampaignClickCounts,
          chartFn: this.makeCharts,
          loading: false,
        },

        bounces: {
          name: this.$t('globals.terms.bounces'),
          type: 'line',
          data: null,
          fn: this.$api.getCampaignBounceCounts,
          chartFn: this.makeCharts,
          donutColor: chartColorRed,
          loading: false,
        },

        links: {
          name: this.$t('analytics.links'),
          type: 'bar',
          data: null,
          chart: null,
          loading: false,
          fn: this.$api.getCampaignLinkCounts,
          chartFn: this.makeLinksChart,
          onClick: this.onLinkClick,
        },
      },

      form: {
        campaigns: [],
        from: null,
        to: null,
      },
    };
  },

  methods: {
    onFromDateChange() {
      if (this.form.from > this.form.to) {
        this.form.to = dayjs(this.form.from).add(7, 'day').toDate();
      }
    },

    onToDateChange() {
      if (this.form.from > this.form.to) {
        this.form.from = dayjs(this.form.to).add(-7, 'day').toDate();
      }
    },

    formatDateTime(s) {
      return dayjs(s).format('YYYY-MM-DD HH:mm');
    },

    isCampaignSelected(camp) {
      return !this.form.campaigns.find(({ id }) => id === camp.id);
    },

    makeLinksChart(typ, camps, data) {
      const labels = data.map((l) => {
        try {
          this.urls.push(l.url);
          const u = new URL(l.url);
          if (l.url.length > 80) {
            return `${u.hostname}${u.pathname.substr(0, 50)}..`;
          }
          return u.hostname + u.pathname;
        } catch {
          return l.url;
        }
      });

      const out = {
        labels,
        datasets: [
          {
            data: data.map((l) => l.count),
            backgroundColor: chartColors,
          }],
      };

      return { points: out, donut: null };
    },

    makeCharts(typ, campaigns, data) {
      // Make a campaign id => camp lookup map to group incoming
      // data by campaigns.
      const camps = campaigns.reduce((obj, c) => {
        const out = { ...obj };
        out[c.id] = c;
        return out;
      }, {});
      const campIDs = Object.keys(camps);
      // datasets[] array for line chart.
      const lines = campIDs.map((id, n) => {
        const cId = parseInt(id, 10);
        const points = data.filter((item) => item.campaignId === cId);

        return {
          label: camps[id].name,
          data: points.map((item) => ({ x: this.formatDateTime(item.timestamp), y: item.count })),
          borderColor: chartColors[n % campIDs.length],
          borderWidth: 2,
          pointHoverBorderWidth: 5,
          pointBorderWidth: 0.5,
        };
      });

      // Donut.
      const labels = [];
      const points = campIDs.map((id) => {
        labels.push(camps[id].name);
        const cId = parseInt(id, 10);
        const sum = data.reduce((a, item) => (item.campaignId === cId ? a + item.count : a), 0);
        return sum;
      });

      const donut = {
        labels,
        datasets: [{
          data: points, backgroundColor: chartColors, borderWidth: 6,
        }],
      };
      return { points: { datasets: lines }, donut };
    },

    onSubmit() {
      this.$router.push({ query: { id: this.form.campaigns.map((c) => c.id), from: dayjs(this.form.from).unix(), to: dayjs(this.form.to).unix() } });
    },

    queryCampaigns(q) {
      this.isSearchLoading = true;
      this.$api.getCampaigns({
        query: q,
        order_by: 'created_at',
        order: 'DESC',
      }).then((data) => {
        this.isSearchLoading = false;
        this.queriedCampaigns = data.results.map((c) => {
          // Change the name to include the ID in the auto-suggest results.
          const camp = c;
          camp.name = `#${c.id}: ${c.name}`;
          return camp;
        });
      });
    },

    getData(typ, camps) {
      this.charts[typ].loading = true;
      // Call the HTTP API.
      this.charts[typ].fn({
        id: camps.map((c) => c.id),
        from: this.form.from,
        to: this.form.to,
      }).then((data) => {
        // Set the total count.
        this.counts[typ] = data.reduce((sum, d) => sum + d.count, 0);

        const { points, donut } = this.charts[typ].chartFn(typ, camps, data);
        this.charts[typ].data = points;
        this.charts[typ].donutData = donut;
        this.charts[typ].loading = false;
      });
    },

    onLinkClick(e) {
      const bars = e.chart.getElementsAtEventForMode(e, 'nearest', { intersect: true }, true);
      if (bars.length > 0) {
        window.open(this.urls[bars[0].index], '_blank', 'noopener noreferrer');
      }
    },
  },

  computed: {
    ...mapState(['serverConfig']),
  },

  created() {
    const now = dayjs().set('hour', 23).set('minute', 59).set('seconds', 0);
    const weekAgo = now.subtract(7, 'day').set('hour', 0).set('minute', 0);
    const from = this.$route.query.from ? dayjs.unix(this.$route.query.from) : weekAgo;
    const to = this.$route.query.to ? dayjs.unix(this.$route.query.to) : now;
    this.form.from = from.toDate();
    this.form.to = to.toDate();
  },

  mounted() {
    // Fetch one or more campaigns if there are ?id params, wait for the fetches
    // to finish, add them to the campaign selector and submit the form.
    const ids = this.$utils.parseQueryIDs(this.$route.query.id);
    if (ids.length > 0) {
      this.isSearchLoading = true;
      Promise.allSettled(ids.map((id) => this.$api.getCampaign(id))).then((data) => {
        data.forEach((d) => {
          if (d.status !== 'fulfilled') {
            return;
          }

          const camp = d.value;
          camp.name = `#${camp.id}: ${camp.name}`;
          this.form.campaigns.push(camp);
        });

        this.$nextTick(() => {
          this.isSearchLoading = false;

          // Fetch count for each analytics type (views, counts, bounces);
          Object.keys(this.charts).forEach((k) => {
            this.charts[k].data = null;
            this.charts[k].donutData = null;

            // Fetch views, clicks, bounces for every campaign.
            this.getData(k, this.form.campaigns);
          });
        });
      });
    }
  },
});
</script>
