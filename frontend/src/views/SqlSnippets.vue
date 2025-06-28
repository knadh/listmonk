<template>
  <section class="sql-snippets">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-4">{{ $t('sqlSnippets.title') }}</h1>
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
            <th>{{ $t('globals.fields.name') }}</th>
            <th>{{ $t('globals.fields.description') }}</th>
            <th>{{ $t('globals.fields.status') }}</th>
            <th>{{ $t('globals.fields.createdAt') }}</th>
            <th>{{ $t('globals.fields.updatedAt') }}</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-if="sqlSnippets.length === 0">
            <td colspan="6" class="has-text-centered">
              {{ $t('globals.messages.emptyState') }}
            </td>
          </tr>
          <tr v-for="snippet in sqlSnippets" :key="snippet.id" :class="{ 'has-text-grey': !snippet.is_active }">
            <td>
              <div>
                <strong>{{ snippet.name }}</strong>
              </div>
            </td>
            <td>{{ snippet.description }}</td>
            <td>
              <b-tag :type="snippet.is_active ? 'is-success' : 'is-light'">
                {{ snippet.is_active ? $t('globals.terms.enabled') : $t('globals.terms.disabled') }}
              </b-tag>
            </td>
            <td>{{ $utils.niceDate(snippet.created_at, true) }}</td>
            <td>{{ $utils.niceDate(snippet.updated_at, true) }}</td>
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
    <b-modal trap-focus :active.sync="form.isVisible" :width="600" scroll="keep">
      <form @submit.prevent="onSubmit">
        <div class="modal-card" style="width: auto">
          <header class="modal-card-head">
            <h4 class="modal-card-title">
              {{ form.id ? $t('globals.buttons.edit') : $t('globals.buttons.new') }}
              {{ $t('sqlSnippets.snippet') }}
            </h4>
          </header>
          <section class="modal-card-body">
            <div class="columns">
              <div class="column">
                <b-field :label="$t('globals.fields.name')" label-position="on-border">
                  <b-input v-model="form.name" name="name" :ref="'focus'" maxlength="200" required />
                </b-field>
              </div>
            </div>

            <div class="columns">
              <div class="column">
                <b-field :label="$t('globals.fields.description')" label-position="on-border">
                  <b-input v-model="form.description" name="description" maxlength="500" type="textarea" />
                </b-field>
              </div>
            </div>

            <div class="columns">
              <div class="column">
                <b-field :label="$t('sqlSnippets.querySQL')" label-position="on-border">
                  <code-editor v-model="form.query_sql" language="sql" :placeholder="$t('sqlSnippets.queryPlaceholder')" />
                </b-field>
                <p class="is-size-7 has-text-grey">
                  {{ $t('sqlSnippets.queryHelp') }}
                </p>
              </div>
            </div>

            <div class="columns">
              <div class="column is-6">
                <b-field>
                  <b-checkbox v-model="form.is_active">{{ $t('globals.fields.status') }}</b-checkbox>
                </b-field>
              </div>
              <div class="column is-6 has-text-right">
                <b-button @click="validateQuery" type="is-info" icon-left="check" :loading="isValidating">
                  {{ $t('sqlSnippets.validate') }}
                </b-button>
              </div>
            </div>

            <div v-if="validationMessage" class="notification" :class="validationMessage.type">
              {{ validationMessage.text }}
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

export default {
  name: 'SqlSnippets',

  components: {
    CodeEditor,
  },

  data() {
    return {
      sqlSnippets: [],
      form: this.initForm(),
      isValidating: false,
      validationMessage: null,
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
        query_sql: '',
        is_active: true,
      };
    },

    showForm(snippet = null) {
      this.form = this.initForm();
      this.validationMessage = null;

      if (snippet) {
        this.form = { ...this.form, ...snippet };
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
      this.$api.createSQLSnippet(this.form).then((data) => {
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
      this.$api.updateSQLSnippet(this.form.id, this.form).then((data) => {
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
        title: this.$t('globals.terms.confirm'),
        message: this.$t('globals.messages.confirmDelete', { name: snippet.name }),
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

    validateQuery() {
      if (!this.form.query_sql.trim()) {
        this.validationMessage = {
          type: 'is-warning',
          text: this.$t('sqlSnippets.emptyQuery'),
        };
        return;
      }

      this.isValidating = true;
      this.validationMessage = null;

      this.$api.validateSQLSnippet({ query_sql: this.form.query_sql }).then(() => {
        this.validationMessage = {
          type: 'is-success',
          text: this.$t('sqlSnippets.validQuery'),
        };
      }).catch((err) => {
        this.validationMessage = {
          type: 'is-danger',
          text: err.message || this.$t('sqlSnippets.invalidQuery'),
        };
      }).finally(() => {
        this.isValidating = false;
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
</style>
