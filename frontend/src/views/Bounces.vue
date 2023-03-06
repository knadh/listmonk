<template>
  <section class="bounces">
    <header class="page-header columns">
      <div class="column is-two-thirds">
        <h1 class="title is-4">{{ $t('globals.terms.bounces') }}
          <span v-if="bounces.total > 0">({{ bounces.total }})</span></h1>
      </div>
      <div class="column has-text-right buttons">
        <b-button v-if="bulk.checked.length > 0 || bulk.all" type="is-primary"
          icon-left="trash-can-outline" data-cy="btn-delete"
          @click.prevent="$utils.confirm(null, () => deleteBounces())">
          {{ $t('globals.buttons.clear') }}
        </b-button>
        <b-button v-if="bounces.total" icon-left="trash-can-outline" data-cy="btn-delete"
          @click.prevent="$utils.confirm(null, () => deleteBounces(true))">
          {{ $t('globals.buttons.clearAll') }}
        </b-button>
      </div>
    </header>

    <b-table :data="bounces.results" :hoverable="true" :loading="loading.bounces"
      default-sort="createdAt"
      checkable
      @check-all="onTableCheck" @check="onTableCheck"
      :checked-rows.sync="bulk.checked"
      detailed
      show-detail-icon
      @details-open="(row) => $buefy.toast.open(`Expanded ${row.user.first_name}`)"
      paginated backend-pagination pagination-position="both" @page-change="onPageChange"
      :current-page="queryParams.page" :per-page="bounces.perPage" :total="bounces.total"
      backend-sorting @sort="onSort">
      <b-table-column v-slot="props" field="email" :label="$t('subscribers.email')"
        :td-attrs="$utils.tdID" sortable>
        <router-link :to="{ name: 'subscriber', params: { id: props.row.subscriberId }}">
          {{ props.row.email }}
        </router-link>
      </b-table-column>

      <b-table-column v-slot="props" field="campaign" :label="$tc('globals.terms.campaign')"
        sortable>
        <router-link v-if="props.row.campaign"
          :to="{ name: 'bounces', query: { campaign_id: props.row.campaign.id }}">
          {{ props.row.campaign.name }}
        </router-link>
        <span v-else>-</span>
      </b-table-column>

      <b-table-column v-slot="props" field="source" :label="$t('bounces.source')" sortable>
        <router-link :to="{ name: 'bounces', query: { source: props.row.source } }">
          {{ props.row.source }}
        </router-link>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at"
        :label="$t('globals.fields.createdAt')" sortable>
        {{ $utils.niceDate(props.row.createdAt, true) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a v-if="!props.row.isDefault" href="#"
            @click.prevent="$utils.confirm(null, () => deleteBounce(props.row))"
            data-cy="btn-delete">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
          <span v-else class="a has-text-grey-light">
              <b-icon icon="trash-can-outline" size="is-small" />
          </span>
        </div>
      </b-table-column>

      <template #detail="props">
        <pre class="is-size-7">{{ props.row.meta }}</pre>
      </template>

      <template #empty v-if="!loading.templates">
        <empty-placeholder />
      </template>
    </b-table>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
  },

  data() {
    return {
      bounces: {},

      // Table bulk row selection states.
      bulk: {
        checked: [],
        all: false,
      },

      // Query params to filter the getSubscribers() API call.
      queryParams: {
        page: 1,
        orderBy: 'created_at',
        order: 'desc',
        campaignID: 0,
        source: '',
      },
    };
  },

  methods: {
    onSort(field, direction) {
      this.queryParams.orderBy = field;
      this.queryParams.order = direction;
      this.getBounces();
    },

    onPageChange(p) {
      this.queryParams.page = p;
      this.getBounces();
    },

    onTableCheck() {
      // Disable bulk.all selection if there are no rows checked in the table.
      if (this.bulk.checked.length !== this.bounces.total) {
        this.bulk.all = false;
      }
    },

    getBounces() {
      this.bulk.checked = [];
      this.bulk.all = false;

      this.$api.getBounces({
        page: this.queryParams.page,
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
        campaign_id: this.queryParams.campaign_id,
        source: this.queryParams.source,
      }).then((data) => {
        this.bounces = data;
      });
    },

    deleteBounce(b) {
      this.$api.deleteBounce(b.id).then(() => {
        this.getBounces();
        this.$utils.toast(this.$t('globals.messages.deleted', { name: b.email }));
      });
    },

    deleteBounces(all) {
      const fnSuccess = () => {
        this.getBounces();
        this.$utils.toast(this.$t('globals.messages.deletedCount',
          { name: this.$tc('globals.terms.bounces'), num: this.bounces.total }));
      };

      if (all) {
        this.$api.deleteBounces({ all: true }).then(fnSuccess);
        return;
      }

      const ids = this.bulk.checked.map((s) => s.id);
      this.$api.deleteBounces({ id: ids }).then(fnSuccess);
    },
  },

  computed: {
    ...mapState(['templates', 'loading']),

    selectedBounces() {
      if (this.bulk.all) {
        return this.bounces.total;
      }
      return this.bulk.checked.length;
    },

  },

  mounted() {
    if (this.$route.query.campaign_id) {
      this.queryParams.campaign_id = parseInt(this.$route.query.campaign_id, 10);
    }

    if (this.$route.query.source) {
      this.queryParams.source = this.$route.query.source;
    }

    this.getBounces();
  },
});
</script>
