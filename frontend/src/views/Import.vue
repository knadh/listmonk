<template>
  <section class="import">
    <h1 class="title is-4">Import subscribers</h1>

    <b-loading :active="isLoading"></b-loading>

    <section v-if="isFree()" class="wrap-small">
      <form @submit.prevent="onSubmit" class="box">
        <div>
          <div class="columns">
            <div class="column">
              <b-field label="Mode">
                <div>
                  <b-radio v-model="form.mode" name="mode"
                    native-value="subscribe">Subscribe</b-radio>
                  <b-radio v-model="form.mode" name="mode"
                    native-value="blocklist">Blocklist</b-radio>
                </div>
              </b-field>
            </div>
            <div class="column">
              <b-field v-if="form.mode === 'subscribe'"
                label="Overwrite?"
                message="Overwrite name and attribs of existing subscribers?">
                <div>
                  <b-switch v-model="form.overwrite" name="overwrite" />
                </div>
              </b-field>
            </div>
            <div class="column">
              <b-field label="CSV delimiter" message="Default delimiter is comma."
                class="delimiter">
                <b-input v-model="form.delim" name="delim"
                  placeholder="," maxlength="1" required />
              </b-field>
            </div>
          </div>

          <list-selector v-if="form.mode === 'subscribe'"
            label="Lists"
            placeholder="Lists to subscribe to"
            message="Lists to subscribe to."
            v-model="form.lists"
            :selected="form.lists"
            :all="lists.results"
          ></list-selector>
          <hr />

          <b-field label="CSV or ZIP file" label-position="on-border">
            <b-upload v-model="form.file" drag-drop expanded>
              <div class="has-text-centered section">
                <p>
                  <b-icon icon="file-upload-outline" size="is-large"></b-icon>
                </p>
                <p>Click or drag a CSV or ZIP file here</p>
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
              :disabled="!form.file || (form.mode === 'subscribe' && form.lists.length === 0)"
              :loading="isProcessing">Upload</b-button>
          </div>
        </div>
      </form>
      <br /><br />

      <div class="import-help">
        <h5 class="title is-size-6">Instructions</h5>
        <p>
          Upload a CSV file or a ZIP file with a single CSV file in it to bulk
          import subscribers. The CSV file should have the following headers
          with the exact column names. <code>attributes</code> (optional)
          should be a valid JSON string with double escaped quotes.
        </p>
        <br />
        <blockquote className="csv-example">
          <code className="csv-headers">
            <span>email,</span>
            <span>name,</span>
            <span>attributes</span>
          </code>
        </blockquote>

        <hr />

        <h5 class="title is-size-6">Example raw CSV</h5>
        <blockquote className="csv-example">
          <code className="csv-headers">
            <span>email,</span>
            <span>name,</span>
            <span>attributes</span>
          </code><br />
          <code className="csv-row">
            <span>user1@mail.com,</span>
            <span>"User One",</span>
            <span>{'"{""age"": 42, ""planet"": ""Mars""}"'}</span>
          </code><br />
          <code className="csv-row">
            <span>user2@mail.com,</span>
            <span>"User Two",</span>
            <span>
              {'"{""age"": 24, ""job"": ""Time Traveller""}"'}
            </span>
          </code>
        </blockquote>
      </div>
    </section><!-- upload //-->

    <section v-if="isRunning() || isDone()" class="wrap status box has-text-centered">
      <b-progress :value="progress" show-value type="is-success"></b-progress>
      <br />
      <p :class="['is-size-5', 'is-capitalized',
          {'has-text-success': status.status === 'finished'},
          {'has-text-danger': (status.status === 'failed' || status.status === 'stopped')}]">
        {{ status.status }}</p>

      <p>{{ status.imported }} / {{ status.total }} records</p>
      <br />

      <p>
        <b-button @click="stopImport" :loading="isProcessing" icon-left="file-upload-outline"
          type="is-primary">{{ isDone() ? 'Done' : 'Stop import' }}</b-button>
      </p>
      <br />

      <p>
        <b-input v-model="logs" id="import-log" class="logs"
          type="textarea" readonly placeholder="Import log" />
      </p>
    </section>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListSelector from '../components/ListSelector.vue';

export default Vue.extend({
  components: {
    ListSelector,
  },

  props: {
    data: {},
    isEditing: null,
  },

  data() {
    return {
      form: {
        mode: 'subscribe',
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

    // Returns true if an import has finished (failed or sucessful).
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
        this.logs = data;

        Vue.nextTick(() => {
          // vue.$refs doesn't work as the logs textarea is rendered dynamiaclly.
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
        delim: this.form.delim,
        lists: this.form.lists.map((l) => l.id),
        overwrite: this.form.overwrite,
      }));
      params.set('file', this.form.file);

      // Post.
      this.$api.importSubscribers(params).then(() => {
        // On file upload, show a confirmation.
        this.$buefy.toast.open({
          message: 'Import started',
          type: 'is-success',
          queue: false,
        });

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
  },
});
</script>
