<template>
  <section class="dashboard content">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-5">{{ dayjs().format("ddd, DD MMM") }}</h1>
      </div>
    </header>

    <section class="counts wrap-small">
      <div class="tile is-ancestor">
        <div class="tile is-vertical is-12">
          <div class="tile">
            <div class="tile is-parent is-vertical relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">{{ $utils.niceNumber(counts.lists.total) }}</p>
                    <p class="is-size-6 has-text-grey">Lists</p>
                  </div>
                  <div class="column is-6">
                    <ul class="no is-size-7 has-text-grey">
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.public) }}</label> public
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.private) }}</label> private
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.optinSingle) }}</label>
                        single opt-in
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.optinDouble) }}</label>
                        double opt-in</li>
                    </ul>
                  </div>
                </div>
              </article><!-- lists -->

              <article class="tile is-child notification">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">{{ $utils.niceNumber(counts.campaigns.total) }}</p>
                    <p class="is-size-6 has-text-grey">Campaigns</p>
                  </div>
                  <div class="column is-6">
                    <ul class="no is-size-7 has-text-grey">
                      <li v-for="(num, status) in counts.campaigns.byStatus" :key="status">
                        <label>{{ num }}</label> {{ status }}
                      </li>
                    </ul>
                  </div>
                </div>
              </article><!-- campaigns -->
            </div><!-- block -->

            <div class="tile is-parent relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">{{ $utils.niceNumber(counts.subscribers.total) }}</p>
                    <p class="is-size-6 has-text-grey">Subscribers</p>
                  </div>

                  <div class="column is-6">
                    <ul class="no is-size-7 has-text-grey">
                      <li>
                        <label>{{ $utils.niceNumber(counts.subscribers.blocklisted) }}</label>
                        blocklisted
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.subscribers.orphans) }}</label>
                        orphans
                      </li>
                    </ul>
                  </div><!-- subscriber breakdown -->
                </div><!-- subscriber columns -->
                <hr />
                <div class="columns">
                  <div class="column is-6">
                    <p class="title">{{ $utils.niceNumber(counts.messages) }}</p>
                    <p class="is-size-6 has-text-grey">Messages sent</p>
                  </div>
                </div>
              </article><!-- subscribers -->
            </div>
          </div>
          <div class="tile is-parent relative">
            <b-loading v-if="isChartsLoading" active :is-full-page="false" />
            <article class="tile is-child notification charts">
              <div class="columns">
                <div class="column is-6">
                  <h3 class="title is-size-6">Campaign views</h3><br />
                  <vue-c3 v-if="chartViewsInst" :handler="chartViewsInst"></vue-c3>
                  <empty-placeholder v-else-if="!isChartsLoading" />
                </div>
                <div class="column is-6">
                  <h3 class="title is-size-6 has-text-right">Link clicks</h3><br />
                  <vue-c3 v-if="chartClicksInst" :handler="chartClicksInst"></vue-c3>
                  <empty-placeholder v-else-if="!isChartsLoading" />
                </div>
              </div>
            </article>
          </div>
        </div>
      </div><!-- tile block -->
    </section>
  </section>
</template>


<style lang="css">
  @import "~c3/c3.css";
</style>

<script>
import Vue from 'vue';
import VueC3 from 'vue-c3';
import dayjs from 'dayjs';
import { colors } from '../constants';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
    VueC3,
  },

  data() {
    return {
      // Unique Vue() instances for each chart.
      chartViewsInst: null,
      chartClicksInst: null,

      isChartsLoading: true,
      isCountsLoading: true,

      counts: {
        lists: {},
        subscribers: {},
        campaigns: {},
        messages: 0,
      },
    };
  },

  methods: {
    makeChart(label, data) {
      const conf = {
        data: {
          columns: [
            [label, ...data.map((d) => d.count).reverse()],
          ],
          type: 'spline',
          color() {
            return colors.primary;
          },
        },
        axis: {
          x: {
            type: 'category',
            categories: data.map((d) => dayjs(d.date).format('DD MMM')).reverse(),
            tick: {
              rotate: -45,
              multiline: false,
              culling: { max: 10 },
            },
          },
        },
        legend: {
          show: false,
        },
      };
      return conf;
    },
  },

  computed: {
    dayjs() {
      return dayjs;
    },
  },

  mounted() {
    // Pull the counts.
    this.$api.getDashboardCounts().then((data) => {
      this.counts = data;
      this.isCountsLoading = false;
    });

    // Pull the charts.
    this.$api.getDashboardCharts().then((data) => {
      this.isChartsLoading = false;

      // vue-c3 lib requires unique instances of Vue() to communicate.
      if (data.campaignViews.length > 0) {
        this.chartViewsInst = this;

        this.$nextTick(() => {
          this.chartViewsInst.$emit('init',
            this.makeChart('Campaign views', data.campaignViews));
        });
      }

      if (data.linkClicks.length > 0) {
        this.chartClicksInst = new Vue();

        this.$nextTick(() => {
          this.chartClicksInst.$emit('init',
            this.makeChart('Link clicks', data.linkClicks));
        });
      }
    });
  },
});
</script>
