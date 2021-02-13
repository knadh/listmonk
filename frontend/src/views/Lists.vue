<template>
  <section class="lists">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-4">
          {{ $t('globals.terms.lists') }}
          <span v-if="!isNaN(lists.total)">({{ lists.total }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="plus" @click="showNewForm">
          {{ $t('globals.buttons.new') }}
        </b-button>
      </div>
    </header>

    <b-table
      :data="lists.results"
      :loading="loading.lists"
      hoverable default-sort="createdAt"
      paginated backend-pagination pagination-position="both" @page-change="onPageChange"
      :current-page="queryParams.page" :per-page="lists.perPage" :total="lists.total"
      backend-sorting @sort="onSort"
    >
        <template slot-scope="props">
            <b-table-column field="name" :label="$t('globals.fields.name')"
              sortable width="25%" paginated backend-pagination pagination-position="both"
              @page-change="onPageChange">
              <div>
                <router-link :to="{name: 'subscribers_list', params: { listID: props.row.id }}">
                  {{ props.row.name }}
                </router-link>
                <b-taglist>
                    <b-tag class="is-small" v-for="t in props.row.tags" :key="t">{{ t }}</b-tag>
                </b-taglist>
              </div>
            </b-table-column>

            <b-table-column field="type" :label="$t('globals.fields.type')" sortable>
              <div>
                <b-tag :class="props.row.type">
                  {{ $t('lists.types.' + props.row.type) }}
                </b-tag>
                {{ ' ' }}
                <b-tag>
                  <b-icon :icon="props.row.optin === 'double' ?
                    'account-check-outline' : 'account-off-outline'" size="is-small" />
                  {{ ' ' }}
                  {{ $t('lists.optins.' + props.row.optin) }}
                </b-tag>{{ ' ' }}
                <a v-if="props.row.optin === 'double'" class="is-size-7 send-optin"
                  href="#" @click="$utils.confirm(null, () => createOptinCampaign(props.row))">
                  <b-tooltip :label="$t('lists.sendOptinCampaign')" type="is-dark">
                    <b-icon icon="rocket-launch-outline" size="is-small" />
                    {{ $t('lists.sendOptinCampaign') }}
                  </b-tooltip>
                </a>
              </div>
            </b-table-column>

            <b-table-column field="subscriber_count" :label="$t('globals.terms.lists')"
              numeric sortable centered>
                <router-link :to="`/subscribers/lists/${props.row.id}`">
                  {{ props.row.subscriberCount }}
                </router-link>
            </b-table-column>

            <b-table-column field="created_at" :label="$t('globals.fields.createdAt')" sortable>
                {{ $utils.niceDate(props.row.createdAt) }}
            </b-table-column>
            <b-table-column field="updated_at" :label="$t('globals.fields.updatedAt')" sortable>
                {{ $utils.niceDate(props.row.updatedAt) }}
            </b-table-column>

            <b-table-column class="actions" align="right">
              <div>
                <router-link :to="`/campaigns/new?list_id=${props.row.id}`">
                  <b-tooltip :label="$t('lists.sendCampaign')" type="is-dark">
                    <b-icon icon="rocket-launch-outline" size="is-small" />
                  </b-tooltip>
                </router-link>
                <a href="" @click.prevent="showEditForm(props.row)">
                  <b-tooltip :label="$t('globals.buttons.edit')" type="is-dark">
                    <b-icon icon="pencil-outline" size="is-small" />
                  </b-tooltip>
                </a>
                <a href="" @click.prevent="deleteList(props.row)">
                  <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
                    <b-icon icon="trash-can-outline" size="is-small" />
                  </b-tooltip>
                </a>
              </div>
            </b-table-column>
        </template>

        <template slot="empty" v-if="!loading.lists">
            <empty-placeholder />
        </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="600">
      <list-form :data="curItem" :isEditing="isEditing" @finished="formFinished"></list-form>
    </b-modal>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListForm from './ListForm.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

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
      queryParams: {
        page: 1,
        orderBy: 'created_at',
        order: 'asc',
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

    getLists() {
      this.$api.getLists({
        page: this.queryParams.page,
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
      });
    },

    deleteList(list) {
      this.$utils.confirm(
        this.$t('lists.confirmDelete'),
        () => {
          this.$api.deleteList(list.id).then(() => {
            this.getLists();

            this.$buefy.toast.open({
              message: this.$t('globals.messages.deleted', { name: list.name }),
              type: 'is-success',
              queue: false,
            });
          });
        },
      );
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
    ...mapState(['loading', 'lists', 'settings']),
  },

  mounted() {
    this.getLists();
  },
});
</script>
