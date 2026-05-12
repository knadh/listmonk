<template>
  <section class="dashboard content">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-5">
          {{ $utils.niceDate(new Date()) }}
        </h1>
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
                        <label for="#">{{ $utils.niceNumber(counts.lists.public) }}</label>
                        {{ $t('lists.types.public') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.private) }}</label>
                        {{ $t('lists.types.private') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.optinSingle) }}</label>
                        {{ $t('lists.optins.single') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.optinDouble) }}</label>
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
                        <label for="#" :data-cy="`campaigns-${status}`">{{ num }}</label>
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
                        <label for="#">{{ $utils.niceNumber(counts.subscribers.blocklisted) }}</label>
                        {{ $t('subscribers.status.blocklisted') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.subscribers.orphans) }}</label>
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
                      {{ $utils.niceNumber(totalMessagesSent) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $t('dashboard.messagesSent') }}
                    </p>
                  </div>
                </div>
              </article><!-- subscribers -->
            </div>
          </div>
          <div class="tile">
            <div class="tile is-parent relative">
              <b-loading v-if="isFeaturesLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="features-left">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="water-outline" />
                      {{ features.drips ? features.drips.total : 0 }}
                    </p>
                    <p class="is-size-6 has-text-grey">Drip Campaigns</p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ features.drips ? features.drips.active : 0 }}</label>
                        Active
                      </li>
                    </ul>
                  </div>
                </div>
                <hr />
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="robot-outline" />
                      {{ features.automations ? features.automations.total : 0 }}
                    </p>
                    <p class="is-size-6 has-text-grey">Automations</p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ features.automations ? features.automations.active : 0 }}</label>
                        Active
                      </li>
                    </ul>
                  </div>
                </div>
                <hr />
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="filter-variant" />
                      {{ features.segments || 0 }}
                    </p>
                    <p class="is-size-6 has-text-grey">Segments</p>
                  </div>
                </div>
              </article>
            </div>
            <div class="tile is-parent relative">
              <b-loading v-if="isFeaturesLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="features-right">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="star-outline" />
                      {{ features.scoring_rules || 0 }}
                    </p>
                    <p class="is-size-6 has-text-grey">Scoring Rules</p>
                  </div>
                </div>
                <hr />
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="handshake-outline" />
                      {{ features.deals ? features.deals.total : 0 }}
                    </p>
                    <p class="is-size-6 has-text-grey">Deals</p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ features.deals ? features.deals.open : 0 }}</label>
                        Open
                      </li>
                    </ul>
                  </div>
                </div>
                <hr />
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="webhook" />
                      {{ features.webhooks ? features.webhooks.total : 0 }}
                    </p>
                    <p class="is-size-6 has-text-grey">Webhooks</p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ features.webhooks ? features.webhooks.active : 0 }}</label>
                        Active
                      </li>
                    </ul>
                  </div>
                </div>
                <hr />
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="fire" />
                      {{ features.warming ? features.warming.total_sent : 0 }}
                    </p>
                    <p class="is-size-6 has-text-grey">Warming Emails Sent</p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ features.warming ? features.warming.campaigns : 0 }}</label>
                        Campaigns
                        ({{ features.warming ? features.warming.active : 0 }} active)
                      </li>
                      <li>
                        <label for="#">{{ features.warming ? features.warming.sent_today : 0 }}</label>
                        Sent today
                      </li>
                      <li>
                        <label for="#">{{ features.warming ? features.warming.total_errors : 0 }}</label>
                        Errors
                      </li>
                      <li>
                        <router-link :to="{ name: 'warmingSendLog' }"
                          class="is-size-7">View send log &rarr;</router-link>
                      </li>
                    </ul>
                  </div>
                </div>
              </article>
            </div>
          </div><!-- features row -->

          <!-- Solomon fork: Campaign Health — surfaces running campaigns and
               flags any whose last send was > 2hr ago as STALLED. The whole
               point is to make a stuck rate-limiter / silent worker stall
               obvious the moment you log in, instead of discovering it days
               later from someone asking "where are the conversions?". -->
          <div class="tile is-parent relative" v-if="health.length > 0">
            <article class="tile is-child notification">
              <h3 class="title is-size-6">
                <b-icon icon="heart-pulse" /> Campaign Health
                <span v-if="anyStalled" class="tag is-danger ml-2">{{ stalledCount }} STALLED</span>
              </h3>
              <b-table :data="health" striped hoverable>
                <b-table-column field="status" label="" v-slot="props" width="40">
                  <b-tag v-if="props.row.stalled" type="is-danger">STALLED</b-tag>
                  <b-tag v-else-if="props.row.idle" type="is-warning is-light">idle</b-tag>
                  <b-tag v-else type="is-success is-light">sending</b-tag>
                </b-table-column>
                <b-table-column field="name" label="Campaign" v-slot="props">
                  <router-link :to="{ name: 'campaign', params: { id: props.row.id } }">{{ props.row.name }}</router-link>
                </b-table-column>
                <b-table-column field="sent" label="Sent / Queued" v-slot="props">
                  {{ props.row.sent.toLocaleString() }} / {{ props.row.to_send.toLocaleString() }}
                </b-table-column>
                <b-table-column field="last_sent_at" label="Last send" v-slot="props">
                  <span v-if="props.row.last_sent_at" :class="{ 'has-text-danger': props.row.stalled }">
                    {{ $utils.niceDate(props.row.last_sent_at, true) }}
                  </span>
                  <span v-else class="has-text-grey">never</span>
                </b-table-column>
                <b-table-column field="rate" label="Send rate" v-slot="props">
                  <span class="has-text-grey">{{ props.row.send_rate || 0 }}/min</span>
                </b-table-column>
              </b-table>
              <p v-if="anyStalled" class="mt-3 is-size-7 has-text-grey">
                A campaign is flagged STALLED when status='running' but the last send is &gt; 2 hours old.
                Open the campaign and try the <strong>Reset window</strong> button. If that doesn't help,
                pause then resume the campaign to spawn a fresh worker pipe.
              </p>
            </article>
          </div>

          <div class="tile is-parent relative">
            <b-loading v-if="isChartsLoading" active :is-full-page="false" />
            <article class="tile is-child notification charts">
              <div class="columns">
                <div class="column is-6">
                  <h3 class="title is-size-6">
                    {{ $t('dashboard.campaignViews') }}
                  </h3><br />
                  <chart type="line" v-if="campaignViews" :data="campaignViews" />
                </div>
                <div class="column is-6">
                  <h3 class="title is-size-6 has-text-right">
                    {{ $t('dashboard.linkClicks') }}
                  </h3><br />
                  <chart type="line" v-if="campaignClicks" :data="campaignClicks" />
                </div>
              </div>
            </article>
          </div>
        </div>
      </div><!-- tile block -->
      <p v-if="settings['app.cache_slow_queries']" class="has-text-grey">
        *{{ $t('globals.messages.slowQueriesCached') }}
        <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferer"
          class="has-text-grey">
          <b-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
        </a>
      </p>
    </section>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';
import { colors } from '../constants';
import Chart from '../components/Chart.vue';

export default Vue.extend({
  components: {
    Chart,
  },

  data() {
    return {
      isChartsLoading: true,
      isCountsLoading: true,
      isFeaturesLoading: true,
      campaignViews: null,
      campaignClicks: null,
      counts: {
        lists: {},
        subscribers: {},
        campaigns: {},
        messages: 0,
      },
      features: {},
      // Solomon fork: per-campaign health rows for the dashboard widget.
      // Each: { id, name, sent, to_send, last_sent_at, send_rate, stalled, idle }
      health: [],
    };
  },

  methods: {
    fetchData() {
      this.isCountsLoading = true;
      this.isChartsLoading = true;

      this.$api.getDashboardCounts().then((data) => {
        this.counts = data;
        this.isCountsLoading = false;
      });

      this.$api.getDashboardCharts().then((data) => {
        this.isChartsLoading = false;
        this.campaignViews = this.makeChart(data.campaignViews);
        this.campaignClicks = this.makeChart(data.linkClicks);
      });

      this.isFeaturesLoading = true;
      this.$api.getDashboardFeatureCounts().then((data) => {
        this.features = data;
        this.isFeaturesLoading = false;
      });

      this.loadHealth();
    },

    // Solomon fork: build the Campaign Health rows. For every running campaign,
    // pull the send-log stats (last_sent_at) and the live send-rate, compute
    // stalled/idle flags, sort STALLED first so the operator sees the problem.
    loadHealth() {
      this.$api.http.get('/api/campaigns?per_page=100')
        .then((res) => {
          const all = ((res.data && res.data.data && res.data.data.results) || []);
          const running = all.filter((c) => c.status === 'running');
          if (running.length === 0) {
            this.health = [];
            return;
          }
          // Build base rows; we'll fill in last_sent_at + send_rate per campaign.
          const rows = running.map((c) => ({
            id: c.id,
            name: c.name,
            sent: c.sent || 0,
            to_send: c.to_send || 0,
            last_sent_at: null,
            send_rate: 0,
            stalled: false,
            idle: false,
          }));

          // Fetch send-rate map (one call covers all running campaigns).
          this.$api.http.get('/api/campaigns/running/stats').then((statsRes) => {
            const list = (statsRes.data && statsRes.data.data) || [];
            const rateById = {};
            list.forEach((s) => { rateById[s.id] = s.send_rate || 0; });
            rows.forEach((_, i) => { rows[i] = { ...rows[i], send_rate: rateById[rows[i].id] || 0 }; });
          }).catch(() => { /* non-fatal */ });

          // Fetch last_sent_at per campaign in parallel. Index-keyed mutation
          // (rather than mutating the row param) keeps eslint no-param-reassign
          // happy.
          const STALL_THRESHOLD_MS = 2 * 60 * 60 * 1000; // 2 hours
          const now = Date.now();
          Promise.all(rows.map((row, i) => this.$api.http.get(`/api/campaigns/${row.id}/send-log/stats`)
            .then((sr) => {
              const stats = (sr.data && sr.data.data) || {};
              const lastSentAt = stats.last_sent_at || null;
              const stalled = lastSentAt
                ? (now - new Date(lastSentAt).getTime()) > STALL_THRESHOLD_MS
                : false;
              const idle = !lastSentAt;
              rows[i] = {
                ...rows[i], last_sent_at: lastSentAt, stalled, idle,
              };
            })
            .catch(() => { /* non-fatal */ }))).then(() => {
            // Stalled rows first, then idle, then sending.
            rows.sort((a, b) => (b.stalled - a.stalled) || (b.idle - a.idle));
            this.health = rows;
          });
        })
        .catch(() => { /* non-fatal — health tile just won't render */ });
    },

    makeChart(data) {
      if (data.length === 0) {
        return {};
      }
      return {
        labels: data.map((d) => dayjs(d.date).format('DD MMM')),
        datasets: [
          {
            data: [...data.map((d) => d.count)],
            borderColor: colors.primary,
            borderWidth: 2,
            pointHoverBorderWidth: 5,
            pointBorderWidth: 0.5,
          },
        ],
      };
    },
  },

  computed: {
    ...mapState(['settings']),
    dayjs() {
      return dayjs;
    },
    totalMessagesSent() {
      const campaignSent = this.counts.messages || 0;
      const warmingSent = this.features.warming
        ? this.features.warming.total_sent || 0 : 0;
      return campaignSent + warmingSent;
    },
    // Solomon fork: how many running campaigns are flagged stalled.
    stalledCount() {
      return this.health.filter((c) => c.stalled).length;
    },
    anyStalled() {
      return this.stalledCount > 0;
    },
  },

  created() {
    this.$root.$on('page.refresh', this.fetchData);
  },

  destroyed() {
    this.$root.$off('page.refresh', this.fetchData);
  },

  mounted() {
    this.fetchData();
  },
});
</script>
