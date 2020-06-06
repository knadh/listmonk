<template>
  <section class="lists">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-4">Lists <span v-if="lists.total > 0">({{ lists.total }})</span></h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="plus" @click="showNewForm">New</b-button>
      </div>
    </header>

    <b-table
      :data="lists.results"
      :loading="loading.lists"
      hoverable
      default-sort="createdAt">
        <template slot-scope="props">
            <b-table-column field="name" label="Name" sortable>
              <router-link :to="{name: 'subscribers_list', params: { listID: props.row.id }}">
                {{ props.row.name }}
              </router-link>
            </b-table-column>

            <b-table-column field="type" label="Type" sortable>
                <b-tag :class="props.row.type">{{ props.row.type }}</b-tag>
                {{ ' ' }}
                <b-tag>
                  <b-icon :icon="props.row.optin === 'double' ?
                    'account-check-outline' : 'account-off-outline'" size="is-small" />
                  {{ ' ' }}
                  {{ props.row.optin }}
                </b-tag>{{ ' ' }}
                <router-link :to="{name: 'campaign', params: {id: 'new'},
                  query: {type: 'optin', 'list_id': props.row.id}}"
                  v-if="props.row.optin === 'double'" class="is-size-7 send-optin">
                  <b-tooltip label="Send opt-in campaign" type="is-dark">
                    <b-icon icon="rocket-launch-outline" size="is-small" />
                    Send opt-in campaign
                  </b-tooltip>
                </router-link>
            </b-table-column>

            <b-table-column field="subscribers" label="Subscribers" numeric sortable centered>
                <router-link :to="`/subscribers/lists/${props.row.id}`">
                  {{ props.row.subscriberCount }}
                </router-link>
            </b-table-column>

            <b-table-column field="createdAt" label="Created" sortable>
                {{ $utils.niceDate(props.row.createdAt) }}
            </b-table-column>
            <b-table-column field="updatedAt" label="Updated" sortable>
                {{ $utils.niceDate(props.row.updatedAt) }}
            </b-table-column>

            <b-table-column class="actions" align="right">
              <router-link :to="`/campaign/new?list_id=${props.row.id}`">
                <b-tooltip label="Send campaign" type="is-dark">
                  <b-icon icon="rocket-launch-outline" size="is-small" />
                </b-tooltip>
              </router-link>
              <a href="" @click.prevent="showEditForm(props.row)">
                <b-tooltip label="Edit" type="is-dark">
                  <b-icon icon="pencil-outline" size="is-small" />
                </b-tooltip>
              </a>
              <a href="" @click.prevent="deleteList(props.row)">
                <b-tooltip label="Delete" type="is-dark">
                  <b-icon icon="trash-can-outline" size="is-small" />
                </b-tooltip>
              </a>
            </b-table-column>
        </template>

        <template slot="empty" v-if="!loading.lists">
            <section class="section">
                <div class="content has-text-grey has-text-centered">
                    <p>
                        <b-icon icon="plus" size="is-large" />
                    </p>
                    <p>Nothing here.</p>
                </div>
            </section>
        </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="450">
      <list-form :data="curItem" :isEditing="isEditing" @finished="formFinished"></list-form>
    </b-modal>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListForm from './ListForm.vue';

Vue.component('list-form', ListForm);

export default Vue.extend({
  components: {
    ListForm,
  },

  data() {
    return {
      // Current list item being edited.
      curItem: null,
      isEditing: false,
      isFormVisible: false,
    };
  },

  methods: {
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
      this.$api.getLists();
    },

    deleteList(list) {
      this.$utils.confirm(
        'Are you sure? This does not delete subscribers.',
        () => {
          this.$api.deleteList(list.id).then(() => {
            this.$api.getLists();

            this.$buefy.toast.open({
              message: `'${list.name}' deleted`,
              type: 'is-success',
              queue: false,
            });
          });
        },
      );
    },
  },

  computed: {
    ...mapState(['lists', 'loading']),
  },

  mounted() {
    this.$api.getLists();
  },
});
</script>
