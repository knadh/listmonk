<template>
  <section class="subscribers">
    <header class="columns">
      <div class="column is-half">
        <h1 class="title is-4">Subscribers
          <span v-if="!isNaN(subscribers.total)">({{ subscribers.total }})</span>
          <span v-if="currentList">
            &raquo; {{ currentList.name }}
          </span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="plus" @click="showNewForm">New</b-button>
      </div>
    </header>

    <section class="subscribers-controls columns">
      <div class="column is-4">
        <form @submit.prevent="querySubscribers">
          <div>
            <b-field grouped>
              <b-input @input="onSimpleQueryInput" v-model="queryInput"
                placeholder="E-mail or name" icon="magnify" ref="query"
                :disabled="isSearchAdvanced"></b-input>
              <b-button native-type="submit" type="is-primary" icon-left="magnify"
                :disabled="isSearchAdvanced"></b-button>
            </b-field>

            <p>
              <a href="#" @click.prevent="toggleAdvancedSearch">
                <b-icon icon="cog-outline" size="is-small" /> Advanced</a>
            </p>

            <div v-if="isSearchAdvanced">
              <b-field>
                <b-input v-model="queryParams.queryExp"
                  @keydown.native.enter="onAdvancedQueryEnter"
                  type="textarea" ref="queryExp"
                  placeholder="subscribers.name LIKE '%user%' or subscribers.status='blocklisted'">
                </b-input>
              </b-field>
              <b-field>
                <span class="is-size-6 has-text-grey">
                  Partial SQL expression to query subscriber attributes.{{ ' ' }}
                  <a href="https://listmonk.app/docs/querying-and-segmentation"
                    target="_blank" rel="noopener noreferrer"> Learn more.
                  </a>
                </span>
              </b-field>

              <div class="buttons">
                <b-button native-type="submit" type="is-primary"
                  icon-left="magnify">Query</b-button>
                <b-button @click.prevent="toggleAdvancedSearch" icon-left="cancel">Reset</b-button>
              </div>
            </div><!-- advanced query -->
          </div>
        </form>
      </div><!-- search -->

      <div class="column is-4 subscribers-bulk" v-if="bulk.checked.length > 0">
        <div>
          <p>
            <span class="is-size-5 has-text-weight-semibold">
              {{ numSelectedSubscribers }} subscriber(s) selected
            </span>
            <span v-if="!bulk.all && subscribers.total > subscribers.perPage">
              &mdash; <a href="" @click.prevent="selectAllSubscribers">
                Select all {{ subscribers.total }}</a>
            </span>
          </p>

          <p class="actions">
            <a href='' @click.prevent="showBulkListForm">
              <b-icon icon="format-list-bulleted-square" size="is-small" /> Manage lists
            </a>

            <a href='' @click.prevent="deleteSubscribers">
              <b-icon icon="trash-can-outline" size="is-small" /> Delete
            </a>

            <a href='' @click.prevent="blocklistSubscribers">
              <b-icon icon="account-off-outline" size="is-small" /> Blocklist
            </a>
          </p><!-- selection actions //-->
        </div>
      </div>
    </section><!-- control -->

    <b-table
      :data="subscribers.results"
      :loading="loading.subscribers"
      @check-all="onTableCheck" @check="onTableCheck"
      :checked-rows.sync="bulk.checked"
      paginated backend-pagination pagination-position="both" @page-change="onPageChange"
      :current-page="queryParams.page" :per-page="subscribers.perPage" :total="subscribers.total"
      hoverable
      checkable>
        <template slot-scope="props">
            <b-table-column field="status" label="Status">
              <a :href="`/subscribers/${props.row.id}`"
                @click.prevent="showEditForm(props.row)">
                <b-tag :class="props.row.status">{{ props.row.status }}</b-tag>
              </a>
            </b-table-column>

            <b-table-column field="email" label="E-mail">
              <a :href="`/subscribers/${props.row.id}`"
                @click.prevent="showEditForm(props.row)">
                {{ props.row.email }}
              </a>
              <b-taglist>
                  <router-link :to="`/subscribers/lists/${props.row.id}`">
                    <b-tag :class="l.subscriptionStatus" v-for="l in props.row.lists"
                      size="is-small" :key="l.id">
                        {{ l.name }} <sup>{{ l.subscriptionStatus }}</sup>
                    </b-tag>
                  </router-link>
              </b-taglist>
            </b-table-column>

            <b-table-column field="name" label="Name">
              <a :href="`/subscribers/${props.row.id}`"
                @click.prevent="showEditForm(props.row)">
                {{ props.row.name }}
              </a>
            </b-table-column>

            <b-table-column field="lists" label="Lists" numeric centered>
              {{ listCount(props.row.lists) }}
            </b-table-column>

            <b-table-column field="createdAt" label="Created">
                {{ $utils.niceDate(props.row.createdAt) }}
            </b-table-column>

            <b-table-column field="updatedAt" label="Updated">
                {{ $utils.niceDate(props.row.updatedAt) }}
            </b-table-column>

            <b-table-column class="actions" align="right">
              <div>
                <a :href="`/api/subscribers/${props.row.id}/export`">
                  <b-tooltip label="Download data" type="is-dark">
                    <b-icon icon="cloud-download-outline" size="is-small" />
                  </b-tooltip>
                </a>
                <a :href="`/subscribers/${props.row.id}`"
                  @click.prevent="showEditForm(props.row)">
                  <b-tooltip label="Edit" type="is-dark">
                    <b-icon icon="pencil-outline" size="is-small" />
                  </b-tooltip>
                </a>
                <a href='' @click.prevent="deleteSubscriber(props.row)">
                  <b-tooltip label="Delete" type="is-dark">
                    <b-icon icon="trash-can-outline" size="is-small" />
                  </b-tooltip>
                </a>
              </div>
            </b-table-column>
        </template>
        <template slot="empty" v-if="!loading.subscribers">
          <empty-placeholder />
        </template>
    </b-table>

    <!-- Manage list modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isBulkListFormVisible" :width="450">
      <subscriber-bulk-list :numSubscribers="this.numSelectedSubscribers"
        @finished="bulkChangeLists" />
    </b-modal>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="600">
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
      },
    };
  },

  methods: {
    // Count the lists from which a subscriber has not unsubscribed.
    listCount(lists) {
      return lists.reduce((defVal, item) => (defVal + item.status !== 'unsubscribed' ? 1 : 0), 0);
    },

    toggleAdvancedSearch() {
      this.isSearchAdvanced = !this.isSearchAdvanced;

      // Toggling to simple search.
      if (!this.isSearchAdvanced) {
        this.$nextTick(() => {
          this.queryInput = '';
          this.queryParams.queryExp = '';
          this.queryParams.page = 1;
          this.$refs.query.focus();

          this.querySubscribers();
        });
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

    sortSubscribers(field, order, event) {
      console.log(field, order, event);
    },

    onPageChange(p) {
      this.queryParams.page = p;
      this.querySubscribers();
    },

    // Prepares an SQL expression for simple name search inputs and saves it
    // in this.queryExp.
    onSimpleQueryInput(v) {
      const q = v.replace(/'/, "''").trim();
      this.queryParams.queryExp = `(name ~* '${q}' OR email ~* '${q}')`;
    },

    // Ctrl + Enter on the advanced query searches.
    onAdvancedQueryEnter(e) {
      if (e.ctrlKey) {
        this.querySubscribers();
      }
    },

    // Search / query subscribers.
    querySubscribers() {
      this.$api.getSubscribers({
        list_id: this.queryParams.listID,
        query: this.queryParams.queryExp,
        page: this.queryParams.page,
      }).then(() => {
        this.bulk.checked = [];
      });
    },

    deleteSubscriber(sub) {
      this.$utils.confirm(
        'Are you sure?',
        () => {
          this.$api.deleteSubscriber(sub.id).then(() => {
            this.querySubscribers();

            this.$buefy.toast.open({
              message: `'${sub.name}' deleted.`,
              type: 'is-success',
              queue: false,
            });
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
            list_ids: [],
          }).then(() => this.querySubscribers());
        };
      }

      this.$utils.confirm(
        `Blocklist ${this.numSelectedSubscribers} subscriber(s)?`,
        fn,
      );
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

              this.$buefy.toast.open({
                message: `${this.numSelectedSubscribers} subscriber(s) deleted`,
                type: 'is-success',
                queue: false,
              });
            });
        };
      } else {
        // 'All' is selected, delete by query.
        fn = () => {
          this.$api.deleteSubscribersByQuery({
            query: this.queryParams.queryExp,
            list_ids: [],
          }).then(() => {
            this.querySubscribers();

            this.$buefy.toast.open({
              message: `${this.numSelectedSubscribers} subscriber(s) deleted`,
              type: 'is-success',
              queue: false,
            });
          });
        };
      }

      this.$utils.confirm(
        `Delete ${this.numSelectedSubscribers} subscriber(s)?`,
        fn,
      );
    },

    bulkChangeLists(action, lists) {
      const data = {
        action,
        query: this.fullQueryExp,
        target_list_ids: lists.map((l) => l.id),
      };

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
        this.$buefy.toast.open({
          message: 'List change applied',
          type: 'is-success',
          queue: false,
        });
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

    // Get subscribers on load.
    this.querySubscribers();
  },
});
</script>
