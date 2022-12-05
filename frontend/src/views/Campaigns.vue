<template>
  <section class="campaigns">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">{{ $t('globals.terms.campaigns') }}
          <span v-if="!isNaN(campaigns.total)">({{ campaigns.total }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field expanded>
          <b-button expanded :to="{name: 'campaign', params:{id: 'new'}}"
            tag="router-link" class="btn-new"
            type="is-primary" icon-left="plus" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <b-table
      :data="campaigns.results"
      :loading="loading.campaigns"
      :row-class="highlightedRow"
      paginated backend-pagination pagination-position="both" @page-change="onPageChange"
      :current-page="queryParams.page" :per-page="campaigns.perPage" :total="campaigns.total"
      hoverable backend-sorting @sort="onSort">

      <template #top-left>
        <div class="columns">
          <div class="column is-6">
            <form @submit.prevent="getCampaigns">
              <div>
                <b-field>
                  <b-input v-model="queryParams.query" name="query" expanded
                    :placeholder="$t('campaigns.queryPlaceholder')" icon="magnify" ref="query" />
                  <p class="controls">
                    <b-button native-type="submit" type="is-primary" icon-left="magnify" />
                  </p>
                </b-field>
              </div>
            </form>
          </div>
        </div>
      </template>

      <b-table-column v-slot="props" cell-class="status" field="status"
        :label="$t('globals.fields.status')" width="10%" sortable
        :td-attrs="$utils.tdID" header-class="cy-status">
        <div>
          <p>
            <router-link :to="{ name: 'campaign', params: { 'id': props.row.id }}">
              <b-tag :class="props.row.status">
                {{ $t(`campaigns.status.${props.row.status}`) }}
              </b-tag>
              <span class="spinner is-tiny" v-if="isRunning(props.row.id)">
                <b-loading :is-full-page="false" active />
              </span>
            </router-link>
          </p>
          <p v-if="isSheduled(props.row)">
            <b-tooltip :label="$t('scheduled')" type="is-dark">
              <span class="is-size-7 has-text-grey scheduled">
                <b-icon icon="alarm" size="is-small" />
                <span v-if="!isDone(props.row) && !isRunning(props.row)">
                  {{ $utils.duration(new Date(), props.row.sendAt, true) }}
                  <br />
                </span>
                {{ $utils.niceDate(props.row.sendAt, true) }}
              </span>
            </b-tooltip>
          </p>
        </div>
      </b-table-column>
      <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')" width="25%"
        sortable header-class="cy-name">
        <div>
          <p>
            <b-tag v-if="props.row.type === 'optin'" class="is-small">
              {{ $t('lists.optin') }}
            </b-tag>
            <router-link :to="{ name: 'campaign', params: { 'id': props.row.id }}">
              {{ props.row.name }}</router-link>
          </p>
          <p class="is-size-7 has-text-grey">{{ props.row.subject }}</p>
          <b-taglist>
              <b-tag class="is-small" v-for="t in props.row.tags" :key="t">{{ t }}</b-tag>
          </b-taglist>
        </div>
      </b-table-column>
      <b-table-column v-slot="props" cell-class="lists" field="lists"
        :label="$t('globals.terms.lists')" width="15%">
        <ul>
          <li v-for="l in props.row.lists" :key="l.id">
            <router-link :to="{name: 'subscribers_list', params: { listID: l.id }}">
              {{ l.name }}
            </router-link>
          </li>
        </ul>
      </b-table-column>
      <b-table-column v-slot="props" field="created_at" :label="$t('campaigns.timestamps')"
        width="19%" sortable header-class="cy-timestamp">
        <div class="fields timestamps" :set="stats = getCampaignStats(props.row)">
          <p>
            <label>{{ $t('globals.fields.createdAt') }}</label>
            <span>{{ $utils.niceDate(props.row.createdAt, true) }}</span>
          </p>
          <p v-if="stats.startedAt">
            <label>{{ $t('campaigns.startedAt') }}</label>
            <span>{{ $utils.niceDate(stats.startedAt, true) }}</span>
          </p>
          <p v-if="isDone(props.row)">
            <label>{{ $t('campaigns.ended') }}</label>
            <span>{{ $utils.niceDate(stats.updatedAt, true) }}</span>
          </p>
          <p v-if="stats.startedAt && stats.updatedAt"
            class="is-capitalized">
            <label><b-icon icon="alarm" size="is-small" /></label>
            <span>{{ $utils.duration(stats.startedAt, stats.updatedAt) }}</span>
          </p>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="stats" :label="$t('campaigns.stats')" width="15%">
        <div class="fields stats" :set="stats = getCampaignStats(props.row)">
          <p>
            <label>{{ $t('campaigns.views') }}</label>
            <span>{{ $utils.formatNumber(props.row.views) }}</span>
          </p>
          <p>
            <label>{{ $t('campaigns.clicks') }}</label>
            <span>{{ $utils.formatNumber(props.row.clicks) }}</span>
          </p>
          <p>
            <label>{{ $t('campaigns.sent') }}</label>
            <span>
              {{ $utils.formatNumber(stats.sent) }} /
              {{ $utils.formatNumber(stats.toSend) }}
            </span>
          </p>
          <p>
            <label>{{ $t('globals.terms.bounces') }}</label>
            <span>
              <router-link :to="{name: 'bounces', query: { campaign_id: props.row.id }}">
                {{ $utils.formatNumber(props.row.bounces) }}
              </router-link>
            </span>
          </p>
          <p v-if="stats.rate">
            <label><b-icon icon="speedometer" size="is-small"></b-icon></label>
            <span class="send-rate">
              <b-tooltip
                :label="`${stats.netRate} / ${$t('campaigns.rateMinuteShort')} @
                  ${$utils.duration(stats.startedAt, stats.updatedAt)}`"
                type="is-dark">
                {{ stats.rate.toFixed(0) }} / {{ $t('campaigns.rateMinuteShort') }}
              </b-tooltip>
            </span>
          </p>
          <p v-if="isRunning(props.row.id)">
            <label>{{ $t('campaigns.progress') }}
              <span class="spinner is-tiny">
                <b-loading :is-full-page="false" active />
              </span>
            </label>
            <span>
              <b-progress :value="stats.sent / stats.toSend * 100" size="is-small" />
            </span>
          </p>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" width="15%" align="right">
        <div>
          <!-- start / pause / resume / scheduled -->
          <a href="" v-if="canStart(props.row)"
            @click.prevent="$utils.confirm(null,
              () => changeCampaignStatus(props.row, 'running'))" data-cy="btn-start">
            <b-tooltip :label="$t('campaigns.start')" type="is-dark">
              <b-icon icon="rocket-launch-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="" v-if="canPause(props.row)"
            @click.prevent="$utils.confirm(null,
              () => changeCampaignStatus(props.row, 'paused'))" data-cy="btn-pause">
            <b-tooltip :label="$t('campaigns.pause')" type="is-dark">
              <b-icon icon="pause-circle-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="" v-if="canResume(props.row)"
            @click.prevent="$utils.confirm(null,
              () => changeCampaignStatus(props.row, 'running'))" data-cy="btn-resume">
            <b-tooltip :label="$t('campaigns.send')" type="is-dark">
              <b-icon icon="rocket-launch-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="" v-if="canSchedule(props.row)"
            @click.prevent="$utils.confirm($t('campaigns.confirmSchedule'),
              () => changeCampaignStatus(props.row, 'scheduled'))" data-cy="btn-schedule">
            <b-tooltip :label="$t('campaigns.schedule')" type="is-dark">
              <b-icon icon="clock-start" size="is-small" />
            </b-tooltip>
          </a>

          <!-- placeholder for finished campaigns -->
          <a v-if="!canCancel(props.row)
            && !canSchedule(props.row) && !canStart(props.row)" data-disabled>
            <b-icon icon="rocket-launch-outline" size="is-small" />
          </a>

          <a href="" v-if="canCancel(props.row)"
            @click.prevent="$utils.confirm(null,
              () => changeCampaignStatus(props.row, 'cancelled'))"
              data-cy="btn-cancel">
            <b-tooltip :label="$t('globals.buttons.cancel')" type="is-dark">
              <b-icon icon="cancel" size="is-small" />
            </b-tooltip>
          </a>
          <a v-else data-disabled>
            <b-icon icon="cancel" size="is-small" />
          </a>

          <a href="" @click.prevent="previewCampaign(props.row)" data-cy="btn-preview">
            <b-tooltip :label="$t('campaigns.preview')" type="is-dark">
              <b-icon icon="file-find-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="" @click.prevent="$utils.prompt($t('globals.buttons.clone'),
              { placeholder: $t('globals.fields.name'),
                value: $t('campaigns.copyOf', { name: props.row.name }) },
                (name) => cloneCampaign(name, props.row))"
              data-cy="btn-clone">
            <b-tooltip :label="$t('globals.buttons.clone')" type="is-dark">
              <b-icon icon="file-multiple-outline" size="is-small" />
            </b-tooltip>
          </a>
          <router-link :to="{ name: 'campaignAnalytics', query: { 'id': props.row.id }}">
            <b-tooltip :label="$t('globals.terms.analytics')" type="is-dark">
              <b-icon icon="chart-bar" size="is-small" />
            </b-tooltip>
          </router-link>
          <a href=""
            @click.prevent="$utils.confirm($t('campaigns.confirmDelete', { name: props.row.name }),
            () => deleteCampaign(props.row))" data-cy="btn-delete">
              <b-icon icon="trash-can-outline" size="is-small" />
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.campaigns">
        <empty-placeholder />
      </template>
    </b-table>

    <campaign-preview v-if="previewItem"
      type='campaign'
      :id="previewItem.id"
      :title="previewItem.name"
      @close="closePreview"></campaign-preview>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';
import CampaignPreview from '../components/CampaignPreview.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    CampaignPreview,
    EmptyPlaceholder,
  },

  data() {
    return {
      previewItem: null,
      queryParams: {
        page: 1,
        query: '',
        orderBy: 'created_at',
        order: 'desc',
      },
      pollID: null,
      campaignStatsData: {},
    };
  },

  methods: {
    // Campaign statuses.
    canStart(c) {
      return c.status === 'draft' && !c.sendAt;
    },
    canSchedule(c) {
      return c.status === 'draft' && c.sendAt;
    },
    canPause(c) {
      return c.status === 'running';
    },
    canCancel(c) {
      return c.status === 'running' || c.status === 'paused';
    },
    canResume(c) {
      return c.status === 'paused';
    },
    isSheduled(c) {
      return c.status === 'scheduled' || c.sendAt !== null;
    },
    isDone(c) {
      return c.status === 'finished' || c.status === 'cancelled';
    },

    isRunning(id) {
      if (id in this.campaignStatsData) {
        return true;
      }
      return false;
    },

    highlightedRow(data) {
      if (data.status === 'running') {
        return ['running'];
      }
      return '';
    },

    onPageChange(p) {
      this.queryParams.page = p;
      this.getCampaigns();
    },

    onSort(field, direction) {
      this.queryParams.orderBy = field;
      this.queryParams.order = direction;
      this.getCampaigns();
    },

    // Campaign actions.
    previewCampaign(c) {
      this.previewItem = c;
    },

    closePreview() {
      this.previewItem = null;
    },

    getCampaigns() {
      this.$api.getCampaigns({
        page: this.queryParams.page,
        query: this.queryParams.query,
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
      });
    },

    // Stats returns the campaign object with stats (sent, toSend etc.)
    // if there's live stats available for running campaigns. Otherwise,
    // it returns the incoming campaign object that has the static stats
    // values.
    getCampaignStats(c) {
      if (c.id in this.campaignStatsData) {
        return this.campaignStatsData[c.id];
      }
      return c;
    },

    pollStats() {
      // Clear any running status polls.
      clearInterval(this.pollID);

      // Poll for the status as long as the import is running.
      this.pollID = setInterval(() => {
        this.$api.getCampaignStats().then((data) => {
          // Stop polling. No running campaigns.
          if (data.length === 0) {
            clearInterval(this.pollID);

            // There were running campaigns and stats earlier. Clear them
            // and refetch the campaigns list with up-to-date fields.
            if (Object.keys(this.campaignStatsData).length > 0) {
              this.getCampaigns();
              this.campaignStatsData = {};
            }
          } else {
            // Turn the list of campaigns [{id: 1, ...}, {id: 2, ...}] into
            // a map indexed by the id: {1: {}, 2: {}}.
            this.campaignStatsData = data.reduce((obj, cur) => ({ ...obj, [cur.id]: cur }), {});
          }
        }, () => {
          clearInterval(this.pollID);
        });
      }, 1000);
    },

    changeCampaignStatus(c, status) {
      this.$api.changeCampaignStatus(c.id, status).then(() => {
        this.$utils.toast(this.$t('campaigns.statusChanged', { name: c.name, status }));
        this.getCampaigns();
        this.pollStats();
      });
    },

    cloneCampaign(name, c) {
      const now = this.$utils.getDate();
      const sendLater = !!c.sendAt;
      let sendAt = null;
      if (sendLater) {
        sendAt = dayjs(c.sendAt).isAfter(now) ? c.sendAt : now.add(7, 'day');
      }


      const data = {
        name,
        subject: c.subject,
        lists: c.lists.map((l) => l.id),
        type: c.type,
        from_email: c.fromEmail,
        content_type: c.contentType,
        messenger: c.messenger,
        tags: c.tags,
        template_id: c.templateId,
        body: c.body,
        altbody: c.altbody,
        headers: c.headers,
        send_later: sendLater,
        send_at: sendAt,
        archive: c.archive,
        archive_template_id: c.archiveTemplateId,
        archive_meta: c.archiveMeta,
      };

      this.$api.createCampaign(data).then((d) => {
        this.$router.push({ name: 'campaign', params: { id: d.id } });
      });
    },

    deleteCampaign(c) {
      this.$api.deleteCampaign(c.id).then(() => {
        this.getCampaigns();
        this.$utils.toast(this.$t('globals.messages.deleted', { name: c.name }));
      });
    },
  },

  computed: {
    ...mapState(['campaigns', 'loading']),
  },

  mounted() {
    this.getCampaigns();
    this.pollStats();
  },

  destroyed() {
    clearInterval(this.pollID);
  },
});
</script>
