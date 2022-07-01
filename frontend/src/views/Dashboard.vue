<template>
  <section class="dashboard content">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-5">{{ $utils.niceDate(new Date()) }}</h1>
      </div>
    </header>

    <section class="counts wrap">
      <div class="tile is-ancestor">
        <div class="tile is-vertical is-12">
          <div class="tile">
            <div class="tile is-parent is-vertical relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="lists">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="format-list-bulleted-square" />
                      {{ $utils.niceNumber(counts.lists.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.list', counts.lists.total) }}
                    </p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.public) }}</label>
                        {{ $t('lists.types.public') }}
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.private) }}</label>
                        {{ $t('lists.types.private') }}
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.optinSingle) }}</label>
                        {{ $t('lists.optins.single') }}
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.lists.optinDouble) }}</label>
                        {{ $t('lists.optins.double') }}
                      </li>
                    </ul>
                  </div>
                </div>
              </article><!-- lists -->

              <article class="tile is-child notification" data-cy="campaigns">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="rocket-launch-outline" />
                      {{ $utils.niceNumber(counts.campaigns.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.campaign', counts.campaigns.total) }}
                    </p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li v-for="(num, status) in counts.campaigns.byStatus" :key="status">
                        <label :data-cy="`campaigns-${status}`">{{ num }}</label>
                        {{ $t(`campaigns.status.${status}`) }}
                        <span v-if="status === 'running'" class="spinner is-tiny">
                          <b-loading :is-full-page="false" active />
                        </span>
                      </li>
                    </ul>
                  </div>
                </div>
              </article><!-- campaigns -->
            </div><!-- block -->

            <div class="tile is-parent relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="subscribers">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="account-multiple" />
                      {{ $utils.niceNumber(counts.subscribers.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.subscriber', counts.subscribers.total) }}
                    </p>
                  </div>

                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label>{{ $utils.niceNumber(counts.subscribers.blocklisted) }}</label>
                        {{ $t('subscribers.status.blocklisted') }}
                      </li>
                      <li>
                        <label>{{ $utils.niceNumber(counts.subscribers.orphans) }}</label>
                        {{ $t('dashboard.orphanSubs') }}
                      </li>
                    </ul>
                  </div><!-- subscriber breakdown -->
                </div><!-- subscriber columns -->
                <hr />
                <div class="columns" data-cy="messages">
                  <div class="column is-12">
                    <p class="title">
                      <b-icon icon="email-outline" />
                      {{ $utils.niceNumber(counts.messages) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $t('dashboard.messagesSent') }}
                    </p>
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
                  <h3 class="title is-size-6">{{ $t('dashboard.campaignViews') }}</h3><br />
                  <div ref="chart-views"></div>
                </div>
                <div class="column is-6">
                  <h3 class="title is-size-6 has-text-right">
                    {{ $t('dashboard.linkClicks') }}
                  </h3><br />
                  <div ref="chart-clicks"></div>
                </div>
              </div>
              <div class="columns">
                <div class="column">
                  <div class="columns is-vcentered">
                    <div class="column">
                      <h3 class="title is-size-6">
                        {{ $t('dashboard.subscribersCount') }}
                      </h3>
                    </div>
                    <div class="column has-text-right">
                      <b-dropdown
                        aria-role="list"
                        class="has-text-left"
                        v-model="currentList"
                        @change="getSubscriberCount"
                        >
                        <template #trigger="{ active }">
                          <b-button
                            :label="currentList.name"
                            :icon-right="active ? 'menu-up' : 'menu-down'" />
                        </template>

                        <template>
                          <b-dropdown-item
                            v-for="list in this.lists"
                            :key="list.id"
                            :value="list"
                            aria-role="listitem"
                          >
                            {{ list.name }}
                          </b-dropdown-item>
                        </template>
                      </b-dropdown>
                    </div>
                  </div>
                  <div ref="chart-subscribers"></div>
                </div>
              </div>
              <div class="columns">
                <div class="column">
                  <h3 class="title is-size-6">
                    {{ $t('dashboard.subscriberDomains') }}
                  </h3><br />
                  <div class="columns">
                    <div class="column">
                      <div ref="chart-domains"></div>
                    </div>
                    <div class="column is-8">
                      <div class="legend-container"></div>
                    </div>
                  </div>
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
  .legend span {
    display: inline-block;
    margin-left: 7px;
    margin-right: 7px;
    padding: 5px;
  }

  .legend .hidden {
    color: rgb(161, 161, 161);
  }

  .legend .hidden span {
    visibility: hidden;
  }

  .legend {
    display: flex;
    flex-direction: column;
    flex-wrap: wrap;
    overflow: auto;
    height: 320px;
    cursor: pointer;
  }

  .legend:hover > div {
    opacity: 50%;
  }

  .legend:hover > div:hover {
    opacity: 100%;
  }
</style>

<script>
import Vue from 'vue';
import c3 from 'c3';
import * as d3 from 'd3';
import dayjs from 'dayjs';
import { colors } from '../constants';

export default Vue.extend({
  data() {
    return {
      isChartsLoading: true,
      isCountsLoading: true,

      counts: {
        lists: {},
        subscribers: {},
        campaigns: {},
        messages: 0,
      },
      lists: [],
      currentList: {
        id: -1,
        name: this.$t('globals.messages.emptyState'),
      },
    };
  },

  methods: {
    renderChart(label, data, el) {
      const conf = {
        bindto: el,
        unload: true,
        data: {
          type: 'spline',
          columns: [],
          color() {
            return colors.primary;
          },
          empty: { label: { text: this.$t('globals.messages.emptyState') } },
        },
        axis: {
          x: {
            type: 'category',
            categories: data.map((d) => dayjs(d.date).format('DD MMM')),
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

      if (data.length > 0) {
        conf.data.columns.push([label, ...data.map((d) => d.count)]);
      }

      this.$nextTick(() => {
        c3.generate(conf);
      });
    },

    renderTimeseriesChart(label, data, el) {
      const dates = ['x'];
      const counts = [label];

      dates.push(...data.map((d) => d.date));
      counts.push(...data.map((d) => d.count));

      const conf = {
        bindto: el,
        unload: true,
        data: {
          x: 'x',
          columns: [
            dates,
            counts,
          ],
          color() {
            return colors.primary;
          },
          empty: { label: { text: this.$t('globals.messages.emptyState') } },
        },
        axis: {
          x: {
            type: 'timeseries',
            tick: {
              format: ((d) => dayjs(d).format('DD MMM')),
              rotate: -45,
              multiline: false,
              culling: { max: 20 },
            },
          },
        },
        legend: {
          show: false,
        },
      };


      this.$nextTick(() => {
        c3.generate(conf);
      });
    },

    renderPieChart(data, el) {
      const conf = {
        bindto: el,
        unload: true,
        data: {
          type: 'pie',
          labels: 'true',
          columns: [...data.map((d) => [d.domain, d.count])],
          empty: { label: { text: this.$t('globals.messages.emptyState') } },
        },
        tooltip: {
          format: {
            value: (value) => `${value} addresses`,
          },
        },
        legend: {
          show: false,
        },
      };

      this.$nextTick(() => {
        const chart = c3.generate(conf);

        d3.select('.legend-container')
          .insert('div', '.chart')
          .attr('class', 'legend')
          .selectAll('div')
          .data([...data.map((d) => d.domain)])
          .enter()
          .append('div')
          .attr('data-id', (id) => id)
          .attr('class', 'is-size-6')
          .html((id) => `<span style="background-color: ${chart.color(id)}"></span> ${id}`)
          .on('mouseover', (event) => {
            chart.focus(event.target.dataset.id);
          })
          .on('mouseout', () => {
            chart.revert();
          })
          .on('click', (event) => {
            event.target.classList.toggle('hidden');
            chart.toggle(event.target.dataset.id);
          });
      });
    },

    getSubscriberCount(list) {
      this.$api.getDashboardSubscriberCounts(list.id).then((data) => {
        this.renderTimeseriesChart(this.$t('dashboard.subscribersCount'), data, this.$refs['chart-subscribers']);
      });
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
      this.renderChart(this.$t('dashboard.campaignViews'), data.campaignViews, this.$refs['chart-views']);
      this.renderChart(this.$t('dashboard.linkClicks'), data.linkClicks, this.$refs['chart-clicks']);
      this.renderPieChart(data.domains, this.$refs['chart-domains']);
    });

    this.$api.getLists({ minimal: true, per_page: 'all' }).then((data) => {
      this.lists = data.results;
      if (this.lists.length > 0) {
        [this.currentList] = this.lists;
        this.getSubscriberCount(this.lists[0]);
      }
    });
  },
});
</script>
