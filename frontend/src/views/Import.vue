<template>
  <section class="import">
    <h1>
      {{ $t('import.title') }}
    </h1>
    <oat-loading :active="isLoading" />

    <section v-if="isFree()" class="wrap">
      <form @submit.prevent="onUpload" class="card">
        <div>
          <div class="row">
            <div class="col-12">
              <oat-field :label="$t('import.mode')" :addons="false">
                <div>
                  <oat-radio v-model="form.mode" name="mode" native-value="subscribe" data-cy="check-subscribe">
                    {{ $t('import.subscribe') }}
                  </oat-radio>
                  <br />
                  <oat-radio v-model="form.mode" name="mode" native-value="blocklist" data-cy="check-blocklist">
                    {{ $t('import.blocklist') }}
                  </oat-radio>
                </div>
              </oat-field>
            </div>
            <div class="col-12">
              <oat-field :label="$t('globals.fields.status')" :addons="false">
                <template v-if="form.mode === 'subscribe'">
                  <oat-radio v-model="form.subStatus" name="subStatus" native-value="unconfirmed"
                    data-cy="check-unconfirmed">
                    {{ $t('subscribers.status.unconfirmed') }}
                  </oat-radio>
                  <oat-radio v-model="form.subStatus" name="subStatus" native-value="confirmed" data-cy="check-confirmed">
                    {{ $t('subscribers.status.confirmed') }}
                  </oat-radio>
                </template>

                <oat-radio v-else v-model="form.subStatus" name="subStatus" native-value="unsubscribed"
                  data-cy="check-unsubscribed">
                  {{ $t('subscribers.status.unsubscribed') }}
                </oat-radio>
              </oat-field>
            </div>

            <div class="col-12">
              <oat-field :label="$t('import.csvDelim')" :message="$t('import.csvDelimHelp')" class="delimiter">
                <input aria-label="field" v-model="form.delim" name="delim" placeholder="," maxlength="1" required>
              </oat-field>
            </div>
          </div>

          <div class="row">
            <div class="col-4">
              <oat-field v-if="form.mode === 'subscribe'" :label="$t('import.overwriteUserInfo')"
                :message="$t('import.overwriteUserInfoHelp')">
                <div>
                  <oat-switch v-model="form.overwriteUserInfo" name="overwriteUserInfo" data-cy="overwrite-user-info" />
                </div>
              </oat-field>
            </div>

            <div class="col-12">
              <oat-field v-if="form.mode === 'subscribe'" :label="$t('import.overwriteSubStatus')"
                :message="$t('import.overwriteSubStatusHelp')">
                <div>
                  <oat-switch v-model="form.overwriteSubStatus" name="overwriteSubStatus"
                    data-cy="overwrite-sub-status" />
                </div>
              </oat-field>
            </div>
          </div>

          <list-selector v-if="form.mode === 'subscribe'" :label="$t('globals.terms.lists')"
            :placeholder="$t('import.listSubHelp')" :message="$t('import.listSubHelp')" v-model="form.lists"
            :selected="form.lists" :all="lists.results" />
          <hr />

          <oat-field :label="$t('import.csvFile')">
            <oat-upload v-model="form.file">
              <div class="align-center app-section">
                <p>
                  <oat-icon icon="file-upload-outline" />
                </p>
                <p>{{ $t('import.csvFileHelp') }}</p>
              </div>
            </oat-upload>
          </oat-field>
          <div class="hstack" v-if="form.file">
            <span closable @close="clearFile">
              {{ form.file.name }}
            </span>
          </div>
          <div class="hstack">
            <button type="submit" data-variant="primary"
              :disabled="!form.file || (form.mode === 'subscribe' && form.lists.length === 0)" :loading="isProcessing">
              {{ $t('import.upload') }}
            </button>
          </div>
        </div>
      </form>
      <br /><br />

      <div class="import-help">
        <h5>
          {{ $t('import.instructions') }}
        </h5>
        <p>{{ $t('import.instructionsHelp') }}</p>
        <br />
        <blockquote class="csv-example">
          <code class="csv-headers"> <span>email,</span> <span>name,</span> <span>attributes</span></code>
        </blockquote>

        <hr />

        <h5>
          {{ $t('import.csvExample') }}
        </h5>

        <pre class="csv-example" v-text="example" />
      </div>
    </section><!-- upload //-->

    <section v-if="isRunning() || isDone()" class="wrap status card align-center">
      <progress :value="progress" data-variant="primary" />
      <br />
      <p
        :class="['', '', { 'success': status.status === 'finished' }, { 'error': (status.status === 'failed' || status.status === 'stopped') }]">
        {{ status.status }}
      </p>

      <p>{{ $t('import.recordsCount', { num: status.imported, total: status.total }) }}</p>
      <br />

      <p>
        <button type="button" @click="stopImport" :loading="isProcessing" data-variant="primary">
          {{ isDone() ? $t('import.importDone') : $t('import.stopImport') }}
        </button>
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
        overwriteUserInfo: false,
        overwriteSubStatus: false,
        file: null,
        example: '',
      },

      // Initial page load still has to wait for the status API to return
      // to either show the form or the status card.
      isLoading: true,

      isProcessing: false,
      status: { status: '' },
      logs: [],
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
        this.logs = data.split('\n').map((line) => line.replace(/\s+importer\.go:\d+:\s*/, ' *: '));
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

    renderExample() {
      const h = 'email,name,attributes\n'
        + 'user1@mail.com,"User One","{""age"": 42, ""planet"": ""Mars""}"\n'
        + 'user2@mail.com,"User Two","{""age"": 24, ""job"": ""Time Traveller""}"';

      this.example = h;
    },

    resetForm() {
      this.form.mode = 'subscribe';
      this.form.overwriteUserInfo = false;
      this.form.overwriteSubStatus = false;
      this.form.file = null;
      this.form.lists = [];
      this.form.subStatus = 'unconfirmed';
      this.form.delim = ',';
    },

    onUpload() {
      if (this.form.mode === 'subscribe' && this.form.overwriteSubStatus) {
        this.$utils.confirm(this.$t('import.subscribeWarning'), this.onSubmit, this.resetForm);
        return;
      }

      this.onSubmit();
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
        overwrite_userinfo: this.form.overwriteUserInfo,
        overwrite_subscription_status: this.form.overwriteSubStatus,
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
    this.renderExample();
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
