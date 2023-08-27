<template>
  <section class="lists">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          {{ $t('globals.terms.lists') }}
          <span v-if="!isNaN(lists.total)">({{ lists.total }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field expanded>
          <b-button expanded type="is-primary" icon-left="plus" class="btn-new"
            @click="showNewForm" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
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
      <template #top-left>
        <div class="columns">
          <div class="column is-6">
            <form @submit.prevent="getLists">
              <div>
                <b-field>
                  <b-input v-model="queryParams.query" name="query" expanded
                    icon="magnify" ref="query" data-cy="query" />
                  <p class="controls">
                    <b-button native-type="submit" type="is-primary" icon-left="magnify"
                      data-cy="btn-query" />
                  </p>
                </b-field>
              </div>
            </form>
          </div>
        </div>
      </template>

      <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')"
        header-class="cy-name" sortable width="25%"
        paginated backend-pagination pagination-position="both"
        :td-attrs="$utils.tdID"
        @page-change="onPageChange">
        <div>
          <a :href="`/lists/${props.row.id}`"
            @click.prevent="showEditForm(props.row)">
            {{ props.row.name }}
          </a>
          <b-taglist>
              <b-tag class="is-small" v-for="t in props.row.tags" :key="t">{{ t }}</b-tag>
          </b-taglist>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="type" :label="$t('globals.fields.type')"
        header-class="cy-type" sortable width="15%">
        <div class="tags">
          <b-tag :class="props.row.type" :data-cy="`type-${props.row.type}`">
            {{ $t(`lists.types.${props.row.type}`) }}
          </b-tag>
          {{ ' ' }}

          <b-tag :class="props.row.optin" :data-cy="`optin-${props.row.optin}`">
            <b-icon :icon="props.row.optin === 'double' ?
              'account-check-outline' : 'account-off-outline'" size="is-small" />
            {{ ' ' }}
            {{ $t(`lists.optins.${props.row.optin}`) }}
          </b-tag>{{ ' ' }}

          <a v-if="props.row.optin === 'double'" class="is-size-7 send-optin"
            href="#" @click="$utils.confirm(null, () => createOptinCampaign(props.row))"
            data-cy="btn-send-optin-campaign">
            <b-tooltip :label="$t('lists.sendOptinCampaign')" type="is-dark">
              <b-icon icon="rocket-launch-outline" size="is-small" />
              {{ $t('lists.sendOptinCampaign') }}
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="subscriber_count"
        :label="$t('globals.terms.subscribers')" header-class="cy-subscribers"
        numeric sortable centered>
        <router-link :to="`/subscribers/lists/${props.row.id}`">
          {{ $utils.formatNumber(props.row.subscriberCount) }}
        </router-link>
      </b-table-column>

      <b-table-column v-slot="props" field="subscriber_counts"
        header-class="cy-subscribers" width="10%">
        <div class="fields stats">
          <p v-for="(count, status) in filterStatuses(props.row)" :key="status">
            <label>{{ $tc(`subscribers.status.${status}`, count) }}</label>
            <span :class="status">{{ $utils.formatNumber(count) }}</span>
          </p>
        </div>
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
          <router-link :to="`/campaigns/new?list_id=${props.row.id}`" data-cy="btn-campaign">
            <b-tooltip :label="$t('lists.sendCampaign')" type="is-dark">
              <b-icon icon="rocket-launch-outline" size="is-small" />
            </b-tooltip>
          </router-link>

          <a href="" @click.prevent="showEditForm(props.row)" data-cy="btn-edit">
            <b-tooltip :label="$t('globals.buttons.edit')" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </a>

          <router-link :to="{name: 'import', query: { list_id: props.row.id }}"
            data-cy="btn-import">
            <b-tooltip :label="$t('import.title')" type="is-dark">
              <b-icon icon="file-upload-outline" size="is-small" />
            </b-tooltip>
          </router-link>

          <a href="" @click.prevent="deleteList(props.row)" data-cy="btn-delete">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.lists">
          <empty-placeholder />
      </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="600"
      @close="onFormClose">
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
      lists: [],
      queryParams: {
        page: 1,
        query: '',
        orderBy: 'id',
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
        query: this.queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ''),
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
      }).then((resp) => {
        this.lists = resp;
      });
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
