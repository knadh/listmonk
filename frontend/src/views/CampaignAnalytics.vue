<template>
  <section class="analytics content relative">
    <h1 class="title is-4">{{ $t('analytics.title') }}</h1>
    <hr />

    <form @submit.prevent="onSubmit">
      <div class="columns">
        <div class="column is-6">
          <b-field :label="$t('globals.terms.campaigns')" label-position="on-border">
            <b-taginput v-model="form.campaigns" :data="queriedCampaigns" name="campaigns" ellipsis
              icon="tag-outline" :placeholder="$t('globals.terms.campaigns')"
              autocomplete :allow-new="false" :before-adding="isCampaignSelected"
              @typing="queryCampaigns" field="name" :loading="isSearchLoading"></b-taginput>
          </b-field>
        </div>

        <div class="column is-5">
          <div class="columns">
            <div class="column is-6">
              <b-field data-cy="from" :label="$t('analytics.fromDate')" label-position="on-border">
                <b-datetimepicker
                  v-model="form.from"
                  icon="calendar-clock"
                  :timepicker="{ hourFormat: '24' }"
                  :datetime-formatter="formatDateTime" @input="onFromDateChange" />
              </b-field>
            </div>
            <div class="column is-6">
              <b-field data-cy="to" :label="$t('analytics.toDate')" label-position="on-border">
                <b-datetimepicker
                  v-model="form.to"
                  icon="calendar-clock"
                  :timepicker="{ hourFormat: '24' }"
                  :datetime-formatter="formatDateTime" @input="onToDateChange" />
              </b-field>
            </div>
          </div><!-- columns -->
        </div><!-- columns -->

        <div class="column is-1">
          <b-button native-type="submit" type="is-primary" icon-left="magnify"
            :disabled="form.campaigns.length === 0" data-cy="btn-search"></b-button>
        </div>
      </div><!-- columns -->
    </form>

    <p class="is-size-7 mt-2 has-text-grey-light">
      <template v-if="settings['privacy.individual_tracking']">
        {{ $t('analytics.isUnique') }}
      </template>
      <template v-else>{{ $t('analytics.nonUnique') }}</template>
    </p>


    <section class="charts mt-5">
      <div class="chart columns" v-for="(v, k) in charts" :key="k">
        <div class="column is-9">
          <b-loading v-if="v.loading" :active="v.loading" :is-full-page="false" />
          <h4 v-if="v.chart !== null">
            {{ v.name }}
            <span class="has-text-grey-light">({{ $utils.niceNumber(counts[k]) }})</span>
          </h4>
          <div :ref="`chart-${k}`" :id="`chart-${k}`"></div>
        </div>
        <div class="column is-2 donut-container">
          <div :ref="`donut-${k}`" :id="`donut-${k}`" class="donut"></div>
        </div>
      </div>
    </section>
  </section>
</template>

<style lang="css">
  @import "~c3/c3.css";
</style>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import dayjs from 'dayjs';
import c3 from 'c3';
import { colors } from '../constants';

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
      charts: {
        views: {
          name: this.$t('campaigns.views'),
          data: [],
          fn: this.$api.getCampaignViewCounts,
          chart: null,
          chartFn: this.processLines,
          donut: null,
          donutFn: this.renderDonutChart,
          loading: false,
        },

        clicks: {
          name: this.$t('campaigns.clicks'),
          data: [],
          fn: this.$api.getCampaignClickCounts,
          chart: null,
          chartFn: this.processLines,
          donut: null,
          donutFn: this.renderDonutChart,
          loading: false,
        },

        bounces: {
          name: this.$t('globals.terms.bounces'),
          data: [],
          fn: this.$api.getCampaignBounceCounts,
          chart: null,
          chartFn: this.processLines,
          donut: null,
          donutFn: this.renderDonutChart,
          donutColor: chartColorRed,
          loading: false,
        },

        links: {
          name: this.$t('analytics.links'),
          data: [],
          chart: null,
          loading: false,
          fn: this.$api.getCampaignLinkCounts,
          chartFn: this.renderLinksChart,
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
    formatDateTime(s) {
      return dayjs(s).format('YYYY-MM-DD HH:mm');
    },

    isCampaignSelected(camp) {
      return !this.form.campaigns.find(({ id }) => id === camp.id);
    },

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

    renderLineChart(typ, data, el) {
      const conf = {
        bindto: el,
        unload: true,
        data: {
          type: 'spline',
          xs: {},
          columns: [],
          names: [],
          colors: {},
          empty: { label: { text: this.$t('globals.messages.emptyState') } },
        },
        axis: {
          x: {
            type: 'timeseries',
            tick: {
              format: '%Y-%m-%d %H:%M',
            },
          },
        },
        legend: {
          show: false,
        },
      };

      // Add campaign data to the chart.
      data.forEach((c, n) => {
        if (c.data.length === 0) {
          return;
        }

        const x = `x${n + 1}`;
        const d = `data${n + 1}`;

        // data1, data2, datan => x1, x2, xn.
        conf.data.xs[d] = x;

        // Campaign name for each datan.
        conf.data.names[d] = c.name;

        // Dates for each xn.
        conf.data.columns.push([x, ...c.data.map((v) => dayjs(v.timestamp))]);

        // Counts for each datan.
        conf.data.columns.push([d, ...c.data.map((v) => v.count)]);

        // Colours for each datan.
        conf.data.colors[d] = chartColors[n % data.length];
      });

      this.$nextTick(() => {
        if (this.charts[typ].chart) {
          this.charts[typ].chart.destroy();
        }

        this.charts[typ].chart = c3.generate(conf);
      });
    },

    renderDonutChart(typ, camps, data) {
      const conf = {
        bindto: this.$refs[`donut-${typ}`][0],
        unload: true,
        data: {
          type: 'gauge',
          columns: [],
        },
        gauge: {
          width: 15,
          max: 100,
        },
        color: {
          pattern: [],
        },
      };

      conf.gauge.max = camps.reduce((sum, c) => sum + c.sent, 0);
      conf.data.columns.push([this.charts[typ].name, data.reduce((sum, d) => sum + d.count, 0)]);
      conf.color.pattern.push(this.charts[typ].donutColor ?? chartColors[0]);

      this.$nextTick(() => {
        if (this.charts[typ].donut) {
          this.charts[typ].donut.destroy();
        }

        if (conf.gauge.max > 0) {
          this.charts[typ].donut = c3.generate(conf);
        }
      });
    },

    renderLinksChart(typ, camps, data) {
      const conf = {
        bindto: this.$refs[`chart-${typ}`][0],
        unload: true,
        data: {
          type: 'bar',
          x: 'x',
          columns: [],
          color: (c, d) => (typeof (d) === 'object' ? chartColors[d.index % data.length] : chartColors[0]),
          empty: { label: { text: this.$t('globals.messages.emptyState') } },
          onclick: (d) => {
            window.open(data[d.index].url, '_blank', 'noopener noreferrer');
          },
        },
        bar: {
          width: {
            max: 30,
          },
        },
        axis: {
          rotated: true,
          x: {
            type: 'category',
            tick: {
              multiline: false,
            },
          },
        },
      };

      // Add link data to the chart.
      // https://c3js.org/samples/axes_x_tick_rotate.html
      conf.data.columns.push(['x', ...data.map((l) => {
        try {
          const u = new URL(l.url);
          if (l.url.length > 80) {
            return `${u.hostname}${u.pathname.substr(0, 50)}..`;
          }
          return u.hostname + u.pathname;
        } catch {
          return l.url;
        }
      })]);
      conf.data.columns.push([this.$t('analytics.count'), ...data.map((l) => l.count)]);

      this.$nextTick(() => {
        if (this.charts[typ].chart) {
          this.charts[typ].chart.destroy();
        }
        this.charts[typ].chart = c3.generate(conf);
      });
    },

    processLines(typ, camps, data) {
      // Make a campaign id => camp lookup map to group incoming
      // data by campaigns.
      const campIDs = camps.reduce((obj, c) => {
        const out = { ...obj };
        out[c.id] = c;
        return out;
      }, {});

      // Group individual data points per campaign id.
      // {1: [...], 2: [...]}
      const groups = data.reduce((obj, d) => {
        const out = { ...obj };
        if (!(d.campaignId in out)) {
          out[d.campaignId] = [];
        }

        out[d.campaignId].push(d);
        return out;
      }, {});

      Object.keys(groups).forEach((k) => {
        this.charts[typ].data.push({
          name: campIDs[groups[k][0].campaignId].name,
          data: groups[k],
        });
      });

      this.$nextTick(() => {
        this.renderLineChart(typ, this.charts[typ].data, this.$refs[`chart-${typ}`][0]);
      });
    },

    onSubmit() {
      // Fetch count for each analytics type (views, counts, bounces);
      Object.keys(this.charts).forEach((k) => {
        // Clear existing data.
        this.charts[k].data = [];

        // Fetch views, clicks, bounces for every campaign.
        this.getData(k, this.form.campaigns);
      });
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

        this.charts[typ].chartFn(typ, camps, data);

        if (this.charts[typ].donutFn && this.settings['privacy.individual_tracking']) {
          this.charts[typ].donutFn(typ, camps, data);
        }
        this.charts[typ].loading = false;
      });
    },
  },

  computed: {
    ...mapState(['settings']),
  },

  created() {
    const now = dayjs().set('hour', 23).set('minute', 59).set('seconds', 0);
    this.form.to = now.toDate();
    this.form.from = now.subtract(7, 'day').set('hour', 0).set('minute', 0).toDate();
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
          this.onSubmit();
        });
      });
    }
  },
});
</script>
