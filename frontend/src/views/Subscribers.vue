<template>
  <section class="subscribers">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">{{ $t('globals.terms.subscribers') }}
          <span v-if="!isNaN(subscribers.total)">
            (<span data-cy="count">{{ subscribers.total }}</span>)
          </span>
          <span v-if="currentList">
            &raquo; {{ currentList.name }}
          </span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field expanded>
          <b-button expanded type="is-primary" icon-left="plus"
            @click="showNewForm" data-cy="btn-new" class="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <section class="subscribers-controls">
      <div class="columns">
        <div class="column is-4">
          <form @submit.prevent="onSubmit">
            <div>
              <b-field addons>
                <b-input @input="onSimpleQueryInput" v-model="queryInput" expanded
                  :placeholder="$t('subscribers.queryPlaceholder')" icon="magnify" ref="query"
                  :disabled="isSearchAdvanced" data-cy="search"></b-input>
                <p class="controls">
                  <b-button native-type="submit" type="is-primary" icon-left="magnify"
                    :disabled="isSearchAdvanced" data-cy="btn-search"></b-button>
                </p>
              </b-field>

              <div v-if="isSearchAdvanced">
                <b-input v-model="queryParams.queryExp"
                  @keydown.native.enter="onAdvancedQueryEnter"
                  type="textarea" ref="queryExp"
                  placeholder="subscribers.name LIKE '%user%' or subscribers.status='blocklisted'"
                  data-cy="query">
                </b-input>
                <span class="is-size-6 has-text-grey">
                  {{ $t('subscribers.advancedQueryHelp') }}.{{ ' ' }}
                  <a href="https://listmonk.app/docs/querying-and-segmentation"
                    target="_blank" rel="noopener noreferrer">
                    {{ $t('globals.buttons.learnMore') }}.
                  </a>
                </span>
                <div class="buttons">
                  <b-button native-type="submit" type="is-primary"
                    icon-left="magnify" data-cy="btn-query">{{ $t('subscribers.query') }}</b-button>
                  <b-button @click.prevent="toggleAdvancedSearch" icon-left="cancel"
                    data-cy="btn-query-reset">
                    {{ $t('subscribers.reset') }}
                  </b-button>
                </div>
              </div><!-- advanced query -->
            </div>
          </form>
          <div v-if="!isSearchAdvanced" class="toggle-advanced">
            <a href="#" @click.prevent="toggleAdvancedSearch" data-cy="btn-advanced-search">
              <b-icon icon="cog-outline" size="is-small" />
              {{ $t('subscribers.advancedQuery') }}
            </a>
          </div>
        </div><!-- search -->
      </div>
    </section><!-- control -->

    <br />
    <b-table
      :data="subscribers.results"
      :loading="loading.subscribers"
      @check-all="onTableCheck" @check="onTableCheck"
      :checked-rows.sync="bulk.checked"
      paginated backend-pagination pagination-position="both" @page-change="onPageChange"
      :current-page="queryParams.page" :per-page="subscribers.perPage" :total="subscribers.total"
      hoverable checkable backend-sorting @sort="onSort">

        <template #top-left>
          <div class="actions">
            <a class="a" href='' @click.prevent="exportSubscribers"
             data-cy="btn-export-subscribers">
              <b-icon icon="cloud-download-outline" size="is-small" />
              {{ $t('subscribers.export') }}
            </a>
            <template v-if="bulk.checked.length > 0">
              <a class="a" href='' @click.prevent="showBulkListForm" data-cy="btn-manage-lists">
                <b-icon icon="format-list-bulleted-square" size="is-small" /> Manage lists
              </a>
              <a class="a" href='' @click.prevent="deleteSubscribers"
                data-cy="btn-delete-subscribers">
                <b-icon icon="trash-can-outline" size="is-small" /> Delete
              </a>
              <a class="a" href='' @click.prevent="blocklistSubscribers"
                data-cy="btn-manage-blocklist">
                <b-icon icon="account-off-outline" size="is-small" /> Blocklist
              </a>
              <span class="a">
                {{ $t('subscribers.numSelected', { num: numSelectedSubscribers }) }}
                <span v-if="!bulk.all && subscribers.total > subscribers.perPage">
                  &mdash;
                  <a href="" @click.prevent="selectAllSubscribers">
                    {{ $t('subscribers.selectAll', { num: subscribers.total }) }}
                  </a>
                </span>
              </span>
            </template>
          </div>
        </template>

        <b-table-column v-slot="props" field="status" :label="$t('globals.fields.status')"
          header-class="cy-status" :td-attrs="$utils.tdID" sortable>
          <a :href="`/subscribers/${props.row.id}`"
            @click.prevent="showEditForm(props.row)">
            <b-tag :class="props.row.status">
              {{ $t(`subscribers.status.${props.row.status}`) }}
            </b-tag>
          </a>
        </b-table-column>

        <b-table-column v-slot="props" field="email" :label="$t('subscribers.email')"
          header-class="cy-email" sortable>
          <a :href="`/subscribers/${props.row.id}`"
            @click.prevent="showEditForm(props.row)">
            {{ props.row.email }}
          </a>
          <b-taglist>
            <template v-for="l in props.row.lists">
              <router-link :to="`/subscribers/lists/${l.id}`"
                v-bind:key="l.id" style="padding-right:0.5em;">
                <b-tag :class="l.subscriptionStatus" size="is-small" :key="l.id">
                  {{ l.name }}
                  <sup v-if="l.optin === 'double' || l.subscriptionStatus == 'unsubscribed'">
                    {{ $t(`subscribers.status.${l.subscriptionStatus}`) }}
                  </sup>
                </b-tag>
              </router-link>
            </template>
          </b-taglist>
        </b-table-column>

        <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')"
           header-class="cy-name" sortable>
          <a :href="`/subscribers/${props.row.id}`"
            @click.prevent="showEditForm(props.row)">
            {{ props.row.name }}
          </a>
        </b-table-column>

        <b-table-column v-slot="props" field="lists" :label="$t('globals.terms.lists')"
          header-class="cy-lists" centered>
          {{ listCount(props.row.lists) }}
        </b-table-column>

        <b-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')"
          header-class="cy-created_at" sortable>
            {{ $utils.niceDate(props.row.createdAt) }}
        </b-table-column>

        <b-table-column v-slot="props" field="updated_at" :label="$t('globals.fields.updatedAt')"
          header-class="cy-updated_at" sortable>
            {{ $utils.niceDate(props.row.updatedAt) }}
        </b-table-column>

        <b-table-column v-slot="props" cell-class="actions" align="right">
          <div>
            <a :href="`/api/subscribers/${props.row.id}/export`" data-cy="btn-download">
              <b-tooltip :label="$t('subscribers.downloadData')" type="is-dark">
                <b-icon icon="cloud-download-outline" size="is-small" />
              </b-tooltip>
            </a>
            <a :href="`/subscribers/${props.row.id}`"
              @click.prevent="showEditForm(props.row)" data-cy="btn-edit">
              <b-tooltip :label="$t('globals.buttons.edit')" type="is-dark">
                <b-icon icon="pencil-outline" size="is-small" />
              </b-tooltip>
            </a>
            <a href='' @click.prevent="deleteSubscriber(props.row)" data-cy="btn-delete">
              <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
                <b-icon icon="trash-can-outline" size="is-small" />
              </b-tooltip>
            </a>
          </div>
        </b-table-column>

        <template #empty v-if="!loading.subscribers">
          <empty-placeholder />
        </template>
    </b-table>

    <!-- Manage list modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isBulkListFormVisible"
      :width="500" class="has-overflow">
      <subscriber-bulk-list :numSubscribers="this.numSelectedSubscribers"
        @finished="bulkChangeLists" />
    </b-modal>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="800"
      @close="onFormClose">
      <subscriber-form :data="curItem" :isEditing="isEditing"
        @finished="querySubscribers"></subscriber-form>
    </b-modal>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import SubscriberForm from './SubscriberForm.vue';
import SubscriberBulkList from './SubscriberBulkList.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import { uris } from '../constants';

export default Vue.extend({
  components: {
    SubscriberForm,
    SubscriberBulkList,
    EmptyPlaceholder,
  },

  data() {
    return {
      // Current subscriber item being edited.
      curItem: null,
      isSearchAdvanced: false,
      isEditing: false,
      isFormVisible: false,
      isBulkListFormVisible: false,

      // Table bulk row selection states.
      bulk: {
        checked: [],
        all: false,
      },

      queryInput: '',

      // Query params to filter the getSubscribers() API call.
      queryParams: {
        // Search query expression.
        queryExp: '',

        // ID of the list the current subscriber view is filtered by.
        listID: null,
        page: 1,
        orderBy: 'id',
        order: 'desc',
      },
    };
  },

  methods: {
    // Count the lists from which a subscriber has not unsubscribed.
    listCount(lists) {
      return lists.reduce((defVal, item) => (defVal + (item.subscriptionStatus !== 'unsubscribed' ? 1 : 0)), 0);
    },

    toggleAdvancedSearch() {
      this.isSearchAdvanced = !this.isSearchAdvanced;

      // Toggling to simple search.
      if (!this.isSearchAdvanced) {
        this.queryInput = '';
        this.queryParams.queryExp = '';
        this.queryParams.page = 1;
        this.querySubscribers();
        this.$refs.query.focus();
        return;
      }

      // Toggling to advanced search.
      this.$nextTick(() => {
        this.$refs.queryExp.focus();
      });
    },

    // Mark all subscribers in the query as selected.
    selectAllSubscribers() {
      this.bulk.all = true;
    },

    onTableCheck() {
      // Disable bulk.all selection if there are no rows checked in the table.
      if (this.bulk.checked.length !== this.subscribers.total) {
        this.bulk.all = false;
      }
    },

    // Show the edit list form.
    showEditForm(sub) {
      this.curItem = sub;
      this.isFormVisible = true;
      this.isEditing = true;
    },

    // Show the new list form.
    showNewForm() {
      this.curItem = {};
      this.isFormVisible = true;
      this.isEditing = false;
    },

    showBulkListForm() {
      this.isBulkListFormVisible = true;
    },

    onFormClose() {
      if (this.$route.params.id) {
        this.$router.push({ name: 'subscribers' });
      }
    },

    onPageChange(p) {
      this.querySubscribers({ page: p });
    },

    onSort(field, direction) {
      this.querySubscribers({ orderBy: field, order: direction });
    },

    // Prepares an SQL expression for simple name search inputs and saves it
    // in this.queryExp.
    onSimpleQueryInput(v) {
      const q = v.replace(/'/, "''").trim();
      this.queryParams.page = 1;

      if (this.$utils.validateEmail(q)) {
        this.queryParams.queryExp = `email = '${q.toLowerCase()}'`;
      } else {
        this.queryParams.queryExp = `(name ~* '${q}' OR email ~* '${q.toLowerCase()}')`;
      }
    },

    // Ctrl + Enter on the advanced query searches.
    onAdvancedQueryEnter(e) {
      if (e.ctrlKey) {
        this.onSubmit();
      }
    },

    onSubmit() {
      this.querySubscribers({ page: 1 });
    },

    // Search / query subscribers.
    querySubscribers(params) {
      this.queryParams = { ...this.queryParams, ...params };

      this.$nextTick(() => {
        this.$api.getSubscribers({
          list_id: this.queryParams.listID,
          query: this.queryParams.queryExp,
          page: this.queryParams.page,
          order_by: this.queryParams.orderBy,
          order: this.queryParams.order,
        }).then(() => {
          this.bulk.checked = [];
        });
      });
    },

    deleteSubscriber(sub) {
      this.$utils.confirm(
        null,
        () => {
          this.$api.deleteSubscriber(sub.id).then(() => {
            this.querySubscribers();

            this.$utils.toast(this.$t('globals.messages.deleted', { name: sub.name }));
          });
        },
      );
    },

    blocklistSubscribers() {
      let fn = null;
      if (!this.bulk.all && this.bulk.checked.length > 0) {
        // If 'all' is not selected, blocklist subscribers by IDs.
        fn = () => {
          const ids = this.bulk.checked.map((s) => s.id);
          this.$api.blocklistSubscribers({ ids })
            .then(() => this.querySubscribers());
        };
      } else {
        // 'All' is selected, blocklist by query.
        fn = () => {
          this.$api.blocklistSubscribersByQuery({
            query: this.queryParams.queryExp,
            list_ids: this.queryParams.listID ? [this.queryParams.listID] : null,
          }).then(() => this.querySubscribers());
        };
      }

      this.$utils.confirm(this.$t('subscribers.confirmBlocklist', { num: this.numSelectedSubscribers }), fn);
    },

    exportSubscribers() {
      const num = !this.bulk.all && this.bulk.checked.length > 0
        ? this.bulk.checked.length : this.subscribers.total;

      this.$utils.confirm(this.$t('subscribers.confirmExport', { num }), () => {
        const q = new URLSearchParams();
        q.append('query', this.queryParams.queryExp);

        if (this.queryParams.listID) {
          q.append('list_id', this.queryParams.listID);
        }

        // Export selected subscribers.
        if (!this.bulk.all && this.bulk.checked.length > 0) {
          this.bulk.checked.map((s) => q.append('id', s.id));
        }

        document.location.href = `${uris.exportSubscribers}?${q.toString()}`;
      });
    },

    deleteSubscribers() {
      let fn = null;
      if (!this.bulk.all && this.bulk.checked.length > 0) {
        // If 'all' is not selected, delete subscribers by IDs.
        fn = () => {
          const ids = this.bulk.checked.map((s) => s.id);
          this.$api.deleteSubscribers({ id: ids })
            .then(() => {
              this.querySubscribers();

              this.$utils.toast(this.$t('subscribers.subscribersDeleted', { num: this.numSelectedSubscribers }));
            });
        };
      } else {
        // 'All' is selected, delete by query.
        fn = () => {
          this.$api.deleteSubscribersByQuery({
            query: this.queryParams.queryExp,
            list_ids: this.queryParams.listID ? [this.queryParams.listID] : null,
          }).then(() => {
            this.querySubscribers();

            this.$utils.toast(this.$t('subscribers.subscribersDeleted',
              { num: this.numSelectedSubscribers }));
          });
        };
      }

      this.$utils.confirm(this.$t('subscribers.confirmDelete', { num: this.numSelectedSubscribers }), fn);
    },

    bulkChangeLists(action, preconfirm, lists) {
      const data = {
        action,
        query: this.fullQueryExp,
        list_ids: this.queryParams.listID ? [this.queryParams.listID] : null,
        target_list_ids: lists.map((l) => l.id),
      };

      if (preconfirm) {
        data.status = 'confirmed';
      }

      let fn = null;
      if (!this.bulk.all && this.bulk.checked.length > 0) {
        // If 'all' is not selected, perform by IDs.
        fn = this.$api.addSubscribersToLists;
        data.ids = this.bulk.checked.map((s) => s.id);
      } else {
        // 'All' is selected, perform by query.
        data.query = this.queryParams.queryExp;
        fn = this.$api.addSubscribersToListsByQuery;
      }

      fn(data).then(() => {
        this.querySubscribers();
        this.$utils.toast(this.$t('subscribers.listChangeApplied'));
      });
    },
  },

  computed: {
    ...mapState(['subscribers', 'lists', 'loading']),

    numSelectedSubscribers() {
      if (this.bulk.all) {
        return this.subscribers.total;
      }
      return this.bulk.checked.length;
    },

    // Returns the list that the subscribers are being filtered by in.
    currentList() {
      if (!this.queryParams.listID || !this.lists.results) {
        return null;
      }

      return this.lists.results.find((l) => l.id === this.queryParams.listID);
    },
  },

  mounted() {
    if (this.$route.params.listID) {
      this.queryParams.listID = parseInt(this.$route.params.listID, 10);
    }

    if (this.$route.params.id) {
      this.$api.getSubscriber(parseInt(this.$route.params.id, 10)).then((data) => {
        this.showEditForm(data);
      });
    } else {
      // Get subscribers on load.
      this.querySubscribers();
    }
  },
});
</script>
