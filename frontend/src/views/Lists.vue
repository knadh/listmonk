<template>
  <section class="lists">
    <header class="row page-header">
      <div class="col-10">
        <h1 class="mb-2">
          {{ $t('globals.terms.lists') }}
          <span v-if="queryParams.status === 'archived'" class="text-lighter">/ {{ queryParams.status }} </span>
          <span v-if="!isNaN(lists.total)">({{ lists.total }})</span>
        </h1>

        <div class="">
          <router-link v-if="queryParams.status !== 'archived'" :to="{ name: 'lists', query: { status: 'archived' } }">
            {{ $t('globals.buttons.view') }} {{ $t('lists.archived').toLowerCase() }} &rarr;
          </router-link>
          <router-link v-else :to="{ name: 'lists' }">
            {{ $t('globals.buttons.view') }} {{ $t('menu.allLists').toLowerCase() }} &rarr;
          </router-link>
        </div>
      </div>
      <div class="col-12 align-right">
        <oat-field v-if="$can('lists:manage_all')">
          <button type="button" data-variant="primary" class="btn-new" @click="showNewForm" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </button>
        </oat-field>
      </div>
    </header>

    <oat-data-table :data="lists.results" :loading="loading.listsFull" @check-all="onTableCheck" @check="onTableCheck"
      :checked-rows.sync="bulk.checked" default-sort="createdAt" paginated backend-pagination
      @page-change="onPageChange" :current-page="queryParams.page" :per-page="lists.perPage"
      :total="lists.total" checkable backend-sorting @sort="onSort">
      <template #top-left>
        <div class="row">
          <div class="col-6">
            <form @submit.prevent="getLists">
              <oat-field>
                <input aria-label="field" v-model="queryParams.query" name="query" icon="magnify" ref="query" data-cy="query">
                <p class="action-controls">
                  <button type="submit" data-variant="primary" data-cy="btn-query" />
                </p>
              </oat-field>
            </form>
          </div>
        </div>
        <div class="actions" v-if="bulk.checked.length > 0">
          <a class="a" href="#" @click.prevent="deleteLists" data-cy="btn-delete-lists">
            <oat-icon icon="trash-can-outline" /> {{ $t('globals.buttons.delete') }}
          </a>
          <span class="a">
            {{ $tc('globals.messages.numSelected', numSelectedLists, { num: numSelectedLists }) }}
            <span v-if="!bulk.all && lists.total > lists.perPage">
              &mdash;
              <a href="#" @click.prevent="onSelectAll" data-cy="select-all-lists">
                {{ $tc('globals.messages.selectAll', lists.total, { num: lists.total }) }}
              </a>
            </span>
          </span>
        </div>
      </template>

      <oat-table-column v-slot="props" field="name" :label="$t('globals.fields.name')" header-class="cy-name" sortable
        width="25%" paginated backend-pagination :td-attrs="$utils.tdID"
        @page-change="onPageChange">
        <div>
          <a :href="`/lists/${props.row.id}`" @click.prevent="showEditForm(props.row)">
            {{ props.row.name }}
          </a>
          <span class="badge-list hstack gap-1">
            <span class="badge secondary" v-for="t in props.row.tags" :key="t">
              {{ t }}
            </span>
          </span>
        </div>
      </oat-table-column>

      <oat-table-column v-slot="props" field="type" :label="$t('globals.fields.type')" header-class="cy-type" sortable
        width="15%">
        <div class="hstack">
          <oat-badge :type="props.row.type" :data-cy="`type-${props.row.type}`">
            {{ $t(`lists.types.${props.row.type}`) }}
          </oat-badge>
          {{ ' ' }}

          <oat-badge :type="props.row.optin" :data-cy="`optin-${props.row.optin}`">
            <oat-icon :icon="props.row.optin === 'double' ? 'account-check-outline' : 'account-off-outline'"
              />
            {{ ' ' }}
            {{ $t(`lists.optins.${props.row.optin}`) }}
          </oat-badge>{{ ' ' }}

          <a v-if="props.row.optin === 'double'" class=" send-optin" href="#"
            @click="$utils.confirm(null, () => createOptinCampaign(props.row))" data-cy="btn-send-optin-campaign">

              <oat-icon icon="rocket-launch-outline" />
              {{ $t('lists.sendOptinCampaign') }}

          </a>
        </div>
      </oat-table-column>

      <oat-table-column v-slot="props" field="subscriber_count" :label="$t('globals.terms.subscribers')"
        header-class="cy-subscribers" numeric sortable centered>
        <template v-if="$can('subscribers:get_all', 'subscribers:get')">
          <router-link :to="`/subscribers/lists/${props.row.id}`">
            {{ $utils.formatNumber(props.row.subscriberCount) }}
            <span class=" view">{{ $t('globals.buttons.view') }}</span>
          </router-link>
        </template>
        <template v-else>
          {{ $utils.formatNumber(props.row.subscriberCount) }}
        </template>
      </oat-table-column>

      <oat-table-column v-slot="props" field="subscriber_counts" header-class="cy-subscribers" width="10%">
        <div class="field-list stats">
          <p v-for="(count, status) in filterStatuses(props.row)" :key="status">
            <label for="#">{{ $tc(`subscribers.status.${status}`, count) }}</label>
            <router-link :to="`/subscribers/lists/${props.row.id}?subscription_status=${status}`" :class="status">
              {{ $utils.formatNumber(count) }}
            </router-link>
          </p>
        </div>
      </oat-table-column>

      <oat-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')"
        header-class="cy-created_at" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </oat-table-column>
      <oat-table-column v-slot="props" field="updated_at" :label="$t('globals.fields.updatedAt')"
        header-class="cy-updated_at" sortable>
        {{ $utils.niceDate(props.row.updatedAt) }}
      </oat-table-column>

      <oat-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <router-link v-if="$can('campaigns:manage')" :to="`/campaigns/new?list_id=${props.row.id}`"
            data-cy="btn-campaign">
<oat-icon icon="rocket-launch-outline" />
</router-link>

          <a v-if="$can('lists:manage') || $canList(props.row.id, 'list:manage')" href="#"
            @click.prevent="showEditForm(props.row)" data-cy="btn-edit" :aria-label="$t('globals.buttons.edit')">

              <oat-icon icon="pencil-outline" />

          </a>

          <router-link v-if="$can('subscribers:import')" :to="{ name: 'import', query: { list_id: props.row.id } }"
            data-cy="btn-import">
<oat-icon icon="file-upload-outline" />
</router-link>

          <a v-if="$can('lists:manage') || $canList(props.row.id, 'list:manage')" href="#"
            @click.prevent="deleteList(props.row)" data-cy="btn-delete" :aria-label="$t('globals.buttons.delete')">

              <oat-icon icon="trash-can-outline" />

          </a>
        </div>
      </oat-table-column>

      <template #empty v-if="!loading.listsFull">
        <empty-placeholder />
      </template>
</oat-data-table>

    <!-- Add / edit form modal -->
    <oat-modal :active.sync="isFormVisible" :width="600" @close="onFormClose">
      <list-form :data="curItem" :is-editing="isEditing" @finished="formFinished" />
    </oat-modal>

    <p v-if="settings['app.cache_slow_queries']" class="text-light">
      *{{ $t('globals.messages.slowQueriesCached') }}
      <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferer"
        class="text-light">
        <oat-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
      </a>
    </p>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import ListForm from './ListForm.vue';

export default Vue.extend({
  components: {
    ListForm,
    EmptyPlaceholder,
  },

  data() {
    return {
      // Current list item being edited.
      curItem: null,
      isEditing: false,
      isFormVisible: false,
      lists: [],
      queryParams: {
        page: 1,
        query: '',
        orderBy: 'id',
        order: 'asc',
        status: this.$route.query.status || 'active',
      },

      // Table bulk row selection states.
      bulk: {
        checked: [],
        all: false,
      },
    };
  },

  methods: {
    onPageChange(p) {
      this.queryParams.page = p;
      this.getLists();
    },

    onSort(field, direction) {
      this.queryParams.orderBy = field;
      this.queryParams.order = direction;
      this.getLists();
    },

    // Show the edit list form.
    showEditForm(list) {
      this.curItem = list;
      this.isFormVisible = true;
      this.isEditing = true;
    },

    // Show the new list form.
    showNewForm() {
      this.curItem = {};
      this.isFormVisible = true;
      this.isEditing = false;
    },

    formFinished() {
      this.getLists();
    },

    onFormClose() {
      if (this.$route.params.id) {
        this.$router.push({ name: 'lists' });
      }
    },

    filterStatuses(list) {
      const out = { ...list.subscriberStatuses };
      if (list.optin === 'single') {
        delete out.unconfirmed;
        delete out.confirmed;
      }
      return out;
    },

    getLists() {
      this.$api.queryLists({
        page: this.queryParams.page,
        query: this.queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' '),
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
        status: this.queryParams.status,
      }).then((resp) => {
        this.lists = resp;
      });

      // Also fetch the minimal lists for the global store that appears
      // in dropdown menus on other pages like import and campaigns.
      this.$api.getLists({ minimal: true, per_page: 'all', status: 'active' });
    },

    deleteList(list) {
      this.$utils.confirm(
        this.$t('lists.confirmDelete'),
        () => {
          this.$api.deleteList(list.id).then(() => {
            this.getLists();

            this.$utils.toast(this.$t('globals.messages.deleted', { name: list.name }));
          });
        },
      );
    },

    // Mark all lists in the query as selected.
    onSelectAll() {
      this.bulk.all = true;
    },

    onTableCheck() {
      // Disable bulk.all selection if there are no rows checked in the table.
      if (this.bulk.checked.length !== this.lists.total) {
        this.bulk.all = false;
      }
    },

    deleteLists() {
      const name = this.$tc('globals.terms.list', this.numSelectedCampaigns);

      const fn = () => {
        const params = {};
        if (!this.bulk.all && this.bulk.checked.length > 0) {
          // If 'all' is not selected, delete lists by IDs.
          params.id = this.bulk.checked.map((l) => l.id);
        } else {
          // 'All' is selected, delete by query.
          params.query = this.queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' ');
          params.all = this.bulk.all;
        }

        this.$api.deleteLists(params)
          .then(() => {
            this.getLists();
            this.$utils.toast(this.$tc(
              'globals.messages.deletedCount',
              this.numSelectedLists,
              { num: this.numSelectedLists, name },
            ));
          });
      };

      this.$utils.confirm(this.$tc(
        'globals.messages.confirmDelete',
        this.numSelectedLists,
        { num: this.numSelectedLists, name: name.toLowerCase() },
      ), fn);
    },

    createOptinCampaign(list) {
      const data = {
        name: this.$t('lists.optinTo', { name: list.name }),
        subject: this.$t('lists.confirmSub', { name: list.name }),
        lists: [list.id],
        from_email: this.settings['app.from_email'],
        content_type: 'richtext',
        messenger: 'email',
        type: 'optin',
      };

      this.$api.createCampaign(data).then((d) => {
        this.$router.push({ name: 'campaign', hash: '#content', params: { id: d.id } });
      });
      return false;
    },
  },

  computed: {
    ...mapState(['loading', 'settings']),

    numSelectedLists() {
      return this.bulk.all ? this.lists.total : this.bulk.checked.length;
    },
  },

  created() {
    this.$root.$on('page.refresh', this.getLists);
  },

  destroyed() {
    this.$root.$off('page.refresh', this.getLists);
  },

  mounted() {
    if (this.$route.params.id) {
      this.$api.getList(parseInt(this.$route.params.id, 10)).then((data) => {
        this.showEditForm(data);
      });
    } else {
      this.getLists();
    }
  },
});
</script>
