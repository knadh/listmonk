<template>
  <section class="campaigns">
    <header class="row page-header">
      <div class="col-8">
        <h1>
          {{ $t('globals.terms.campaigns') }}
          <span v-if="!isNaN(campaigns.total)">({{ campaigns.total }})</span>
        </h1>
      </div>
      <div class="col-4 col-end align-right">
        <oat-field v-if="$can('campaigns:manage')">
          <button type="button" :to="{ name: 'campaign', params: { id: 'new' } }" tag="router-link" class="btn-new"
            data-variant="primary" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </button>
        </oat-field>
      </div>
    </header>

    <div class="card page-content">
    <oat-data-table :data="campaigns.results" :loading="loading.campaigns" :row-class="highlightedRow"
      @check-all="onTableCheck" @check="onTableCheck" :checked-rows.sync="bulk.checked" paginated backend-pagination
      @page-change="onPageChange" :current-page="queryParams.page"
      :per-page="campaigns.perPage" :total="campaigns.total" checkable backend-sorting @sort="onSort">
      <template #top-left>
        <div class="row">
          <div class="col-6">
            <form @submit.prevent="getCampaigns">
              <fieldset class="group">
                <input aria-label="Search" v-model="queryParams.query" name="query"
                  :placeholder="$t('campaigns.queryPlaceholder')" ref="query">
                <button type="submit" data-variant="primary" aria-label="Search">
                  <oat-icon icon="magnify" />
                </button>
              </fieldset>
            </form>
          </div>
        </div>

        <div class="actions" v-if="bulk.checked.length > 0">
          <a class="a" href="#" @click.prevent="deleteCampaigns" data-cy="btn-delete-campaigns">
            <oat-icon icon="trash-can-outline" /> Delete
          </a>
          <span class="a">
            {{ $tc('globals.messages.numSelected', numSelectedCampaigns, { num: numSelectedCampaigns }) }}
            <span v-if="!bulk.all && campaigns.total > campaigns.perPage">
              &mdash;
              <a href="#" @click.prevent="onSelectAll" data-cy="select-all-campaigns">
                {{ $tc('globals.messages.selectAll', campaigns.total, { num: campaigns.total }) }}
              </a>
            </span>
          </span>
        </div>
      </template>

      <oat-table-column v-slot="props" cell-class="status" field="status" :label="$t('globals.fields.status')" width="10%"
        sortable :td-attrs="$utils.tdID" header-class="cy-status">
        <div>
          <p>
            <router-link :to="{ name: 'campaign', params: { id: props.row.id } }">
              <oat-badge :type="props.row.status">
                {{ $t(`campaigns.status.${props.row.status}`) }}
              </oat-badge>
              <span class="spinner" v-if="isRunning(props.row.id)">
                <oat-loading :is-full-page="false" active />
              </span>
            </router-link>
          </p>
          <p v-if="isSheduled(props.row)">
            <span class="text-light text-7 scheduled">
              <oat-icon icon="alarm" />
              <span v-if="!isDone(props.row) && !isRunning(props.row)">
                {{ $utils.duration(new Date(), props.row.sendAt, true) }}
                <br />
              </span>
              {{ $utils.niceDate(props.row.sendAt, true) }}
            </span>
          </p>
        </div>
      </oat-table-column>
      <oat-table-column v-slot="props" field="name" :label="$t('globals.fields.name')" width="25%" sortable
        header-class="cy-name">
        <div>
          <p>
            <oat-badge v-if="props.row.type === 'optin'" type="optin">
              {{ $t('lists.optin') }}
            </oat-badge>
            <router-link :to="{ name: 'campaign', params: { id: props.row.id } }">
              {{ props.row.name }}
              <copy-text :text="props.row.name" hide-text />
            </router-link>
          </p>
          <p class="text-light text-7">
            <copy-text :text="props.row.subject" />
          </p>
          <span class="badge-list hstack gap-1">
            <span class="badge secondary" v-for="t in props.row.tags" :key="t">
              {{ t }}
            </span>
          </span>
        </div>
      </oat-table-column>
      <oat-table-column v-slot="props" cell-class="lists" field="lists" :label="$t('globals.terms.lists')" width="15%">
        <ul>
          <li v-for="l in props.row.lists" :key="l.id">
            <router-link :to="{ name: 'subscribers_list', params: { listID: l.id } }">
              {{ l.name }}
            </router-link>
          </li>
        </ul>
      </oat-table-column>
      <oat-table-column v-slot="props" field="created_at" :label="$t('campaigns.timestamps')" width="19%" sortable
        header-class="cy-timestamp">
        <div class="field-list timestamps" :set="stats = getCampaignStats(props.row)">
          <p>
            <label for="#">{{ $t('globals.fields.createdAt') }}</label>
            <span>{{ $utils.niceDate(props.row.createdAt, true) }}</span>
          </p>
          <p v-if="stats.startedAt">
            <label for="#">{{ $t('campaigns.startedAt') }}</label>
            <span>{{ $utils.niceDate(stats.startedAt, true) }}</span>
          </p>
          <p v-if="isDone(props.row)">
            <label for="#">{{ $t('campaigns.ended') }}</label>
            <span>{{ $utils.niceDate(stats.updatedAt, true) }}</span>
          </p>
          <p v-if="stats.startedAt && stats.updatedAt" class="capitalize">
            <label for="#"><oat-icon icon="alarm" /></label>
            <span>{{ $utils.duration(stats.startedAt, stats.updatedAt) }}</span>
          </p>
        </div>
      </oat-table-column>

      <oat-table-column v-slot="props" field="stats" :label="$t('campaigns.stats')" width="15%">
        <div class="field-list stats" :set="stats = getCampaignStats(props.row)">
          <p>
            <label for="#">{{ $t('campaigns.views') }}</label>
            <span>{{ $utils.formatNumber(props.row.views) }}</span>
          </p>
          <p>
            <label for="#">{{ $t('campaigns.clicks') }}</label>
            <span>{{ $utils.formatNumber(props.row.clicks) }}</span>
          </p>
          <p>
            <label for="#">{{ $t('campaigns.sent') }}</label>
            <span>
              {{ $utils.formatNumber(stats.sent) }} /
              {{ $utils.formatNumber(stats.toSend) }}
            </span>
          </p>
          <p>
            <label for="#">{{ $t('globals.terms.bounces') }}</label>
            <span>
              <router-link :to="{ name: 'bounces', query: { campaign_id: props.row.id } }">
                {{ $utils.formatNumber(props.row.bounces) }}
              </router-link>
            </span>
          </p>
          <p v-if="stats.rate">
            <label for="#"><oat-icon icon="speedometer" /></label>
            <span class="send-rate">

                {{ stats.rate.toFixed(0) }} / {{ $t('campaigns.rateMinuteShort') }}

            </span>
          </p>
          <p v-if="isRunning(props.row.id)">
            <label for="#">
              {{ $t('campaigns.progress') }}
              <span class="spinner">
                <oat-loading :is-full-page="false" active />
              </span>
            </label>
            <span>
              <progress :value="stats.sent / stats.toSend * 100" />
            </span>
          </p>
        </div>
      </oat-table-column>

      <oat-table-column v-slot="props" cell-class="actions" width="15%" align="right">
        <div>
          <!-- start / pause / resume / scheduled -->
          <template v-if="$can('campaigns:send')">
            <a v-if="canStart(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'running'))"
              data-cy="btn-start" :aria-label="$t('campaigns.start')">

                <oat-icon icon="rocket-launch-outline" />

            </a>

            <a v-if="canPause(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'paused'))" data-cy="btn-pause"
              :aria-label="$t('campaigns.pause')">

                <oat-icon icon="pause-circle-outline" />

            </a>

            <a v-if="canResume(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'running'))"
              data-cy="btn-resume" :aria-label="$t('campaigns.send')">

                <oat-icon icon="rocket-launch-outline" />

            </a>

            <a v-if="canSchedule(props.row)" href="#"
              @click.prevent="$utils.confirm($t('campaigns.confirmSchedule'), () => changeCampaignStatus(props.row, 'scheduled'))"
              data-cy="btn-schedule" :aria-label="$t('campaigns.schedule')">

                <oat-icon icon="clock-start" />

            </a>

            <!-- placeholder for finished campaigns -->
            <a v-if="!canCancel(props.row) && !canSchedule(props.row) && !canStart(props.row)" href="#" data-disabled
              aria-label=" ">
              <oat-icon icon="rocket-launch-outline" />
            </a>

            <a v-if="canCancel(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'cancelled'))"
              data-cy="btn-cancel" :aria-label="$t('globals.buttons.cancel')">

                <oat-icon icon="cancel" />

            </a>
            <a v-else href="#" data-disabled aria-label=" ">
              <oat-icon icon="cancel" />
            </a>
          </template>

          <a href="#" @click.prevent="previewCampaign(props.row)" data-cy="btn-preview"
            :aria-label="$t('campaigns.preview')">

              <oat-icon icon="file-find-outline" />

          </a>
          <a v-if="$can('campaigns:manage')" href="#" @click.prevent="$utils.prompt($t('globals.buttons.clone'),
            {
              placeholder: $t('globals.fields.name'),
              value: $t('campaigns.copyOf', { name: props.row.name }),
            },
            (name) => cloneCampaign(name, props.row))" data-cy="btn-clone" :aria-label="$t('globals.buttons.clone')">

              <oat-icon icon="file-multiple-outline" />

          </a>
          <router-link v-if="$can('campaigns:get_analytics')"
            :to="{ name: 'campaignAnalytics', query: { id: props.row.id } }">
<oat-icon icon="chart-bar" />
</router-link>
          <a v-if="$can('campaigns:manage')" href="#"
            @click.prevent="$utils.confirm($t('campaigns.confirmDelete', { name: props.row.name }), () => deleteCampaign(props.row))"
            data-cy="btn-delete" :aria-label="$t('globals.buttons.delete')">
            <oat-icon icon="trash-can-outline" />
          </a>
        </div>
      </oat-table-column>

      <template #empty v-if="!loading.campaigns">
        <empty-placeholder />
      </template>
</oat-data-table>

    <campaign-preview v-if="previewItem" type="campaign" :id="previewItem.id" :title="previewItem.name"
      @close="closePreview" />
    </div>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';
import CampaignPreview from '../components/CampaignPreview.vue';
import CopyText from '../components/CopyText.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    CampaignPreview,
    EmptyPlaceholder,
    CopyText,
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

      // Table bulk row selection states.
      bulk: {
        checked: [],
        all: false,
      },
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
        query: this.queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' '),
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
        no_body: true,
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

    async cloneCampaign(name, c) {
      // Fetch the template body from the server.
      let body = '';
      let bodySource = null;
      await this.$api.getCampaign(c.id).then((data) => {
        body = data.body;
        bodySource = data.bodySource;
      });

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
        body,
        body_source: bodySource,
        altbody: c.altbody,
        headers: c.headers,
        send_later: sendLater,
        send_at: sendAt,
        archive: c.archive,
        archive_template_id: c.archiveTemplateId,
        archive_meta: c.archiveMeta,
        media: c.media.map((m) => m.id),
      };

      if (c.archive) {
        data.archive_slug = `${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${Date.now().toString().slice(-4)}`;
      }

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

    // Mark all campaigns in the query as selected.
    onSelectAll() {
      this.bulk.all = true;
    },

    onTableCheck() {
      // Disable bulk.all selection if there are no rows checked in the table.
      if (this.bulk.checked.length !== this.campaigns.total) {
        this.bulk.all = false;
      }
    },

    deleteCampaigns() {
      const name = this.$tc('globals.terms.campaign', this.numSelectedCampaigns);

      const fn = () => {
        const params = {};
        if (!this.bulk.all && this.bulk.checked.length > 0) {
          // If 'all' is not selected, delete campaigns by IDs.
          params.id = this.bulk.checked.map((c) => c.id);
        } else {
          // 'All' is selected, delete by query.
          params.query = this.queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' ');
          params.all = this.bulk.all;
        }

        this.$api.deleteCampaigns(params)
          .then(() => {
            this.getCampaigns();
            this.$utils.toast(this.$tc(
              'globals.messages.deletedCount',
              this.numSelectedCampaigns,
              { num: this.numSelectedCampaigns, name },
            ));
          });
      };

      this.$utils.confirm(this.$tc(
        'globals.messages.confirmDelete',
        this.numSelectedCampaigns,
        { num: this.numSelectedCampaigns, name: name.toLowerCase() },
      ), fn);
    },
  },

  computed: {
    ...mapState(['campaigns', 'loading']),

    numSelectedCampaigns() {
      return this.bulk.all ? this.campaigns.total : this.bulk.checked.length;
    },
  },

  created() {
    this.$root.$on('page.refresh', this.getCampaigns);
  },

  mounted() {
    this.getCampaigns();
    this.pollStats();
  },

  destroyed() {
    this.$root.$off('page.refresh', this.getCampaigns);
    clearInterval(this.pollID);
  },
});
</script>
