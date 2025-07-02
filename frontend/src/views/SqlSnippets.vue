<template>
  <section class="sql-snippets">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-4">
          <b-icon icon="code" size="is-small" class="mr-2" />
          {{ $t('sqlSnippets.title') }}
        </h1>
        <p class="has-text-grey">{{ $t('sqlSnippets.description') }}</p>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="plus" @click="showForm" data-cy="btn-new">
          {{ $t('globals.buttons.new') }}
        </b-button>
      </div>
    </header>

    <div class="table-container">
      <table class="table is-fullwidth is-striped is-hoverable">
        <thead>
          <tr>
            <th>
              <b-icon icon="tag-outline" size="is-small" class="mr-1" />
              {{ $t('globals.fields.name') }}
            </th>
            <th>
              <b-icon icon="text" size="is-small" class="mr-1" />
              {{ $t('globals.fields.description') }}
            </th>
            <th>
              <b-icon icon="check-circle-outline" size="is-small" class="mr-1" />
              {{ $t('globals.fields.status') }}
            </th>
            <th>
              <b-icon icon="calendar-clock" size="is-small" class="mr-1" />
              {{ $t('globals.fields.createdAt') }}
            </th>
            <th>
              <b-icon icon="calendar-clock" size="is-small" class="mr-1" />
              {{ $t('globals.fields.updatedAt') }}
            </th>
            <th>
              <b-icon icon="cog-outline" size="is-small" />
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="sqlSnippets.length === 0">
            <td colspan="6" class="has-text-centered has-text-grey py-6">
              <b-icon icon="plus" size="is-large" class="mb-2" />
              <br />
              {{ $t('globals.messages.emptyState') }}
            </td>
          </tr>
          <tr v-for="snippet in sqlSnippets" :key="snippet.id" :class="{ 'has-text-grey': !snippet.isActive }">
            <td>
              <div class="is-flex is-align-items-center">
                <b-icon icon="code" size="is-small" class="mr-2 has-text-info" />
                <strong>{{ snippet.name }}</strong>
              </div>
            </td>
            <td>{{ snippet.description }}</td>
            <td>
              <b-tag :type="snippet.isActive ? 'is-success' : 'is-light'">
                <b-icon :icon="snippet.isActive ? 'check-circle-outline' : 'pause-circle-outline'" size="is-small" class="mr-1" />
                {{ snippet.isActive ? $t('users.status.enabled') : $t('users.status.disabled') }}
              </b-tag>
            </td>
            <td>{{ $utils.niceDate(snippet.createdAt, true) }}</td>
            <td>{{ $utils.niceDate(snippet.updatedAt, true) }}</td>
            <td class="actions">
              <div>
                <b-button @click="showForm(snippet)" icon-left="edit-outline" size="is-small" type="is-text">
                  {{ $t('globals.buttons.edit') }}
                </b-button>
                <b-button @click="deleteSnippet(snippet)" icon-left="trash-can-outline" size="is-small" type="is-text">
                  {{ $t('globals.buttons.delete') }}
                </b-button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- SQL Snippet Form Modal -->
    <b-modal trap-focus :active.sync="form.isVisible" :width="900" scroll="keep">
      <form @submit.prevent="onSubmit">
        <div class="modal-card" style="width: auto">
          <header class="modal-card-head">
            <h4 class="modal-card-title">
              <b-icon :icon="form.id ? 'pencil-outline' : 'plus'" size="is-small" class="mr-2" />
              {{ form.id ? $t('globals.buttons.edit') : $t('globals.buttons.new') }}
              {{ $t('sqlSnippets.snippet') }}
            </h4>
          </header>
          <section class="modal-card-body">
            <div class="columns">
              <div class="column">
                <b-field label-position="on-border">
                  <template #label>
                    <b-icon icon="tag-outline" size="is-small" class="mr-1" />
                    {{ $t('globals.fields.name') }}
                  </template>
                  <b-input v-model="form.name" name="name" :ref="'focus'" maxlength="200" required />
                </b-field>
              </div>
            </div>

            <div class="columns">
              <div class="column">
                <b-field label-position="on-border">
                  <template #label>
                    <b-icon icon="text" size="is-small" class="mr-1" />
                    {{ $t('globals.fields.description') }}
                  </template>
                  <b-input v-model="form.description" name="description" maxlength="500" type="textarea" />
                </b-field>
              </div>
            </div>

            <div class="columns">
              <div class="column">
                <b-field label-position="on-border">
                  <template #label>
                    <b-icon icon="code" size="is-small" class="mr-1" />
                    {{ $t('sqlSnippets.querySQL') }}
                  </template>
                  <code-editor v-model="form.querySql" language="sql" :placeholder="$t('sqlSnippets.queryPlaceholder')" />
                </b-field>
                <p class="is-size-7 has-text-grey">
                  {{ $t('sqlSnippets.queryHelp') }}
                </p>

                <!-- SQL Counter -->
                <SQLCounter
                  :query="form.querySql"
                  :live-validation-enabled.sync="liveValidationEnabled"
                />
              </div>
            </div>

            <div class="columns">
              <div class="column">
                <b-field>
                  <b-checkbox v-model="form.is_active">
                    {{ $t('globals.fields.status') }}
                  </b-checkbox>
                </b-field>
              </div>
            </div>
          </section>
          <footer class="modal-card-foot has-text-right">
            <b-button @click="form.isVisible = false">{{ $t('globals.buttons.cancel') }}</b-button>
            <b-button native-type="submit" type="is-primary" :loading="loading.sqlSnippets">
              {{ $t('globals.buttons.save') }}
            </b-button>
          </footer>
        </div>
      </form>
    </b-modal>
  </section>
</template>

<script>
import CodeEditor from '../components/CodeEditor.vue';
import SQLCounter from '../components/SQLCounter.vue';

export default {
  name: 'SqlSnippets',

  components: {
    CodeEditor,
    SQLCounter,
  },

  data() {
    return {
      sqlSnippets: [],
      form: this.initForm(),
      liveValidationEnabled: true,
    };
  },

  computed: {
    loading() {
      return this.$store.state.loading;
    },
  },

  methods: {
    initForm() {
      return {
        isVisible: false,
        id: 0,
        name: '',
        description: '',
        querySql: '',
        is_active: true,
      };
    },

    showForm(snippet = null) {
      this.form = this.initForm();

      if (snippet) {
        // If editing existing snippet, fetch full data including querySql
        if (snippet.id) {
          this.$api.getSQLSnippet(snippet.id).then((data) => {
            this.form = {
              ...this.form,
              ...data,
              is_active: data.isActive, // Convert camelCase to snake_case for form
              querySql: data.querySql || data.query_sql || '',
            };
            this.form.isVisible = true;
            this.$nextTick(() => {
              this.$refs.focus.focus();
            });
          });
          return;
        }
        // Convert camelCase to snake_case for form
        this.form = {
          ...this.form,
          ...snippet,
          is_active: snippet.isActive,
          querySql: snippet.querySql || snippet.query_sql || '',
        };
      }

      this.form.isVisible = true;
      this.$nextTick(() => {
        this.$refs.focus.focus();
      });
    },

    onSubmit() {
      if (this.form.id) {
        this.updateSnippet();
      } else {
        this.createSnippet();
      }
    },

    createSnippet() {
      const payload = {
        name: this.form.name,
        description: this.form.description,
        query_sql: this.form.querySql,
        is_active: this.form.is_active,
      };
      this.$api.createSQLSnippet(payload).then((data) => {
        this.$buefy.toast.open({
          message: this.$t('globals.messages.created', { name: data.name }),
          type: 'is-success',
          queue: false,
        });

        this.form.isVisible = false;
        this.fetchSnippets();
      });
    },

    updateSnippet() {
      const payload = {
        name: this.form.name,
        description: this.form.description,
        query_sql: this.form.querySql,
        is_active: this.form.is_active,
      };
      this.$api.updateSQLSnippet(this.form.id, payload).then((data) => {
        this.$buefy.toast.open({
          message: this.$t('globals.messages.updated', { name: data.name }),
          type: 'is-success',
          queue: false,
        });

        this.form.isVisible = false;
        this.fetchSnippets();
      });
    },

    deleteSnippet(snippet) {
      this.$buefy.dialog.confirm({
        title: this.$t('globals.messages.confirm'),
        message: this.$t('sqlSnippets.confirmDelete', { name: snippet.name }),
        confirmText: this.$t('globals.buttons.delete'),
        type: 'is-danger',
        onConfirm: () => {
          this.$api.deleteSQLSnippet(snippet.id).then(() => {
            this.$buefy.toast.open({
              message: this.$t('globals.messages.deleted', { name: snippet.name }),
              type: 'is-success',
              queue: false,
            });

            this.fetchSnippets();
          });
        },
      });
    },

    fetchSnippets() {
      this.$api.getSQLSnippets().then((data) => {
        this.sqlSnippets = data;
      });
    },
  },

  mounted() {
    this.fetchSnippets();
    // Load live validation preference
    this.liveValidationEnabled = this.$utils.getPref('sqlSnippets.liveValidation') !== false; // Default to true
  },
};
</script>

<style scoped>
.actions {
  text-align: right;
}

.actions > div {
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

/* Reduce CodeEditor height in modal */
:deep(.code-editor) {
  height: 200px !important;
}
</style>
