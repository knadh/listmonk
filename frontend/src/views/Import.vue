<template>
  <section class="import">
    <h1 class="title is-4">
      {{ $t('import.title') }}
    </h1>
    <b-loading :active="isLoading" />

    <section v-if="isFree()" class="wrap">
      <form @submit.prevent="onSubmit" class="box">
        <div>
          <div class="columns">
            <div class="column">
              <b-field :label="$t('import.mode')" :addons="false">
                <div>
                  <b-radio v-model="form.mode" name="mode" native-value="subscribe" data-cy="check-subscribe">
                    {{ $t('import.subscribe') }}
                  </b-radio>
                  <br />
                  <b-radio v-model="form.mode" name="mode" native-value="blocklist" data-cy="check-blocklist">
                    {{ $t('import.blocklist') }}
                  </b-radio>
                </div>
              </b-field>
            </div>
            <div class="column">
              <b-field :label="$t('globals.fields.status')" :addons="false">
                <template v-if="form.mode === 'subscribe'">
                  <b-radio v-model="form.subStatus" name="subStatus" native-value="unconfirmed"
                    data-cy="check-unconfirmed">
                    {{ $t('subscribers.status.unconfirmed') }}
                  </b-radio>
                  <b-radio v-model="form.subStatus" name="subStatus" native-value="confirmed" data-cy="check-confirmed">
                    {{ $t('subscribers.status.confirmed') }}
                  </b-radio>
                </template>

                <b-radio v-else v-model="form.subStatus" name="subStatus" native-value="unsubscribed"
                  data-cy="check-unsubscribed">
                  {{ $t('subscribers.status.unsubscribed') }}
                </b-radio>
              </b-field>
            </div>

            <div class="column">
              <b-field v-if="form.mode === 'subscribe'" :label="$t('import.overwrite')"
                :message="$t('import.overwriteHelp')">
                <div>
                  <b-switch v-model="form.overwrite" name="overwrite" data-cy="overwrite" />
                </div>
              </b-field>
            </div>

            <div class="column">
              <b-field :label="$t('import.csvDelim')" :message="$t('import.csvDelimHelp')" class="delimiter">
                <b-input v-model="form.delim" name="delim" placeholder="," maxlength="1" required />
              </b-field>
            </div>
          </div>

          <list-selector v-if="form.mode === 'subscribe'" :label="$t('globals.terms.lists')"
            :placeholder="$t('import.listSubHelp')" :message="$t('import.listSubHelp')" v-model="form.lists"
            :selected="form.lists" :all="lists.results" />
          <hr />

          <b-field :label="$t('import.csvFile')" label-position="on-border">
            <b-upload v-model="form.file" drag-drop expanded>
              <div class="has-text-centered section">
                <p>
                  <b-icon icon="file-upload-outline" size="is-large" />
                </p>
                <p>{{ $t('import.csvFileHelp') }}</p>
              </div>
            </b-upload>
          </b-field>
          <div class="tags" v-if="form.file">
            <b-tag size="is-medium" closable @close="clearFile">
              {{ form.file.name }}
            </b-tag>
          </div>
          <div class="buttons">
            <b-button native-type="submit" type="is-primary"
              :disabled="!form.file || (form.mode === 'subscribe' && form.lists.length === 0)" :loading="isProcessing">
              {{ $t('import.upload') }}
            </b-button>
          </div>
        </div>
      </form>
      <br /><br />

      <div class="import-help">
        <h5 class="title is-size-6">
          {{ $t('import.instructions') }}
        </h5>
        <p>{{ $t('import.instructionsHelp') }}</p>
        <br />
        <blockquote className="csv-example">
          <code className="csv-headers">
              <span>email,</span>
              <span>name,</span>
              <span>attributes</span>
            </code>
        </blockquote>

        <hr />

        <h5 class="title is-size-6">
          {{ $t('import.csvExample') }}
        </h5>
        <blockquote className="csv-example">
          <code className="csv-headers">
              <span>email,</span>
              <span>name,</span>
              <span>attributes</span>
            </code><br />
          <code className="csv-row">
              <span>user1@mail.com,</span>
              <span>"User One",</span>
              <span>"{""age"": 42, ""planet"": ""Mars""}"</span>
            </code><br />
          <code className="csv-row">
              <span>user2@mail.com,</span>
              <span>"User Two",</span>
              <span>"{""age"": 24, ""job"": ""Time Traveller""}"</span>
            </code>
        </blockquote>
      </div>
    </section><!-- upload //-->

    <section v-if="isRunning() || isDone()" class="wrap status box has-text-centered">
      <b-progress :value="progress" show-value type="is-success" />
      <br />
      <p
        :class="['is-size-5', 'is-capitalized', { 'has-text-success': status.status === 'finished' }, { 'has-text-danger': (status.status === 'failed' || status.status === 'stopped') }]">
        {{ status.status }}
      </p>

      <p>{{ $t('import.recordsCount', { num: status.imported, total: status.total }) }}</p>
      <br />

      <p>
        <b-button @click="stopImport" :loading="isProcessing" icon-left="file-upload-outline" type="is-primary">
          {{ isDone() ? $t('import.importDone') : $t('import.stopImport') }}
        </b-button>
      </p>
      <br />

      <div class="import-logs">
        <log-view :lines="logs" :loading="false" />
      </div>
    </section>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListSelector from '../components/ListSelector.vue';
import LogView from '../components/LogView.vue';

export default Vue.extend({
  components: {
    ListSelector,
    LogView,
  },

  props: {
    data: { type: Object, default: () => { } },
    isEditing: { type: Boolean, default: false },
  },

  data() {
    return {
      form: {
        mode: 'subscribe',
        subStatus: 'unconfirmed',
        delim: ',',
        lists: [],
        overwrite: true,
        file: null,
      },

      // Initial page load still has to wait for the status API to return
      // to either show the form or the status box.
      isLoading: true,

      isProcessing: false,
      status: { status: '' },
      logs: '',
      pollID: null,
    };
  },

  watch: {
    'form.mode': function formMode() {
      // Select the appropriate status radio whenever mode changes.
      this.$nextTick(() => {
        if (this.form.mode === 'subscribe') {
          this.form.subStatus = 'unconfirmed';
        } else {
          this.form.subStatus = 'unsubscribed';
        }
      });
    },
  },

  methods: {
    clearFile() {
      this.form.file = null;
    },

    // Returns true if we're free to do an upload.
    isFree() {
      if (this.status.status === 'none') {
        return true;
      }
      return false;
    },

    // Returns true if an import is running.
    isRunning() {
      if (this.status.status === 'importing'
        || this.status.status === 'stopping') {
        return true;
      }
      return false;
    },

    isSuccessful() {
      return this.status.status === 'finished';
    },

    isFailed() {
      return (
        this.status.status === 'stopped'
        || this.status.status === 'failed'
      );
    },

    // Returns true if an import has finished (failed or successful).
    isDone() {
      if (this.status.status === 'finished'
        || this.status.status === 'stopped'
        || this.status.status === 'failed'
      ) {
        return true;
      }
      return false;
    },

    pollStatus() {
      // Clear any running status polls.
      clearInterval(this.pollID);

      // Poll for the status as long as the import is running.
      this.pollID = setInterval(() => {
        this.$api.getImportStatus().then((data) => {
          this.isProcessing = false;
          this.isLoading = false;
          this.status = data;
          this.getLogs();

          if (!this.isRunning()) {
            clearInterval(this.pollID);
          }
        }, () => {
          this.isProcessing = false;
          this.isLoading = false;
          this.status = { status: 'none' };
          clearInterval(this.pollID);
        });
        return true;
      }, 250);
    },

    getLogs() {
      this.$api.getImportLogs().then((data) => {
        this.logs = data.split('\n');

        Vue.nextTick(() => {
          // vue.$refs doesn't work as the logs textarea is rendered dynamically.
          const ref = document.getElementById('import-log');
          if (ref) {
            ref.scrollTop = ref.scrollHeight;
          }
        });
      });
    },

    // Cancel a running import or clears a finished import.
    stopImport() {
      this.isProcessing = true;
      this.$api.stopImport().then(() => {
        this.pollStatus();
        this.form.file = null;
      });
    },

    onSubmit() {
      this.isProcessing = true;

      // Prepare the upload payload.
      const params = new FormData();
      params.set('params', JSON.stringify({
        mode: this.form.mode,
        subscription_status: this.form.subStatus,
        delim: this.form.delim,
        lists: this.form.lists.map((l) => l.id),
        overwrite: this.form.overwrite,
      }));
      params.set('file', this.form.file);

      // Post.
      this.$api.importSubscribers(params).then(() => {
        // On file upload, show a confirmation.
        this.$utils.toast(this.$t('import.importStarted'));

        // Start polling status.
        this.pollStatus();
      }, () => {
        this.isProcessing = false;
        this.form.file = null;
      });
    },
  },

  computed: {
    ...mapState(['lists']),

    // Import progress bar value.
    progress() {
      if (!this.status || !this.status.total > 0) {
        return 0;
      }
      return Math.ceil((this.status.imported / this.status.total) * 100);
    },
  },

  mounted() {
    this.pollStatus();

    const ids = this.$utils.parseQueryIDs(this.$route.query.list_id);
    if (ids.length > 0 && this.lists.results) {
      this.$nextTick(() => {
        this.form.lists = this.lists.results.filter((l) => ids.indexOf(l.id) > -1);
      });
    }
  },
});
</script>
