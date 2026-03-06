<template>
  <section class="media-files">
    <h1 class="title is-4">
      {{ $t('media.title') }}
      <span v-if="media.results && media.results.length > 0">({{ media.results.length }})</span>
      <span class="has-text-grey-light"> / {{ serverConfig.media_provider }}</span>
    </h1>

    <b-loading :active="isProcessing || loading.media" />

    <section class="wrap gallery mt-6">
      <div class="columns mb-4">
        <div class="column">
          <form @submit.prevent="onQueryMedia" class="search">
            <div>
              <b-field>
                <b-input v-model="queryParams.query" name="query" expanded icon="magnify" ref="query" data-cy="query" />
                <p class="controls">
                  <b-button native-type="submit" type="is-primary" icon-left="magnify" data-cy="btn-query" />
                </p>
              </b-field>
            </div>
          </form>
        </div>
        <div v-if="$can('media:manage')" class="column is-narrow">
          <b-button @click="onToggleForm" icon-left="file-upload-outline" data-cy="btn-toggle-upload">
            {{ $t('media.upload') }}
          </b-button>
        </div>
      </div>

      <b-collapse v-if="$can('media:manage')" v-model="showUploadForm" animation="">
        <form @submit.prevent="onSubmit" class="mb-6" data-cy="upload">
          <div>
            <b-field :label="$t('media.upload')">
              <b-upload v-model="form.files" drag-drop multiple xaccept=".png,.jpg,.jpeg,.gif,.svg" expanded>
                <div class="has-text-centered section">
                  <p>
                    <b-icon icon="file-upload-outline" size="is-large" />
                  </p>
                  <p>{{ $t('media.uploadHelp') }}</p>
                </div>
              </b-upload>
            </b-field>
            <div class="tags" v-if="form.files.length > 0">
              <b-tag v-for="(f, i) in form.files" :key="i" size="is-medium" closable @close="removeUploadFile(i)">
                {{ f.name }}
              </b-tag>
            </div>
            <div class="buttons">
              <b-button native-type="submit" type="is-primary" icon-left="file-upload-outline"
                :disabled="form.files.length === 0" :loading="isProcessing">
                {{ $tc('media.upload') }}
              </b-button>
            </div>
          </div>
        </form>
      </b-collapse>

      <!-- Pagination -->
      <div v-if="media.total > media.perPage" class="pagination-wrapper mt-5">
        <b-pagination :total="media.total" :current.sync="media.page" :per-page="media.perPage"
          @change="onPageChange" />
      </div>

      <div v-if="loading.media" class="has-text-centered py-6">
        <b-loading :active="loading.media" />
      </div>
      <div v-else-if="media.results && media.results.length > 0" class="grid">
        <div v-for="item in media.results" :key="item.id" class="item">
          <div class="thumb">
            <a @click="(e) => onMediaSelect(item, e)" :href="item.url" target="_blank" rel="noopener noreferer"
              class="thumb-link">
              <div class="thumb-container">
                <img v-if="item.thumbUrl" :src="item.thumbUrl" :title="item.filename" :alt="item.filename" />
                <div v-else class="thumb-placeholder">
                  <span class="file-ext">
                    {{ item.filename.split(".").pop().toUpperCase() }}
                  </span>
                </div>
              </div>
            </a>
            <div class="actions">
              <a href="#" @click.prevent="$utils.confirm(null, () => onDeleteMedia(item.id))" data-cy="btn-delete"
                :aria-label="$t('globals.buttons.delete')" class="delete-btn">
                <b-icon icon="trash-can-outline" size="is-small" />
              </a>
            </div>
          </div>
          <div class="info">
            <p class="filename" :title="item.filename">{{ item.filename }}</p>
            <p class="date">{{ $utils.niceDate(item.createdAt, false) }}</p>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else-if="!loading.media">
        <empty-placeholder />
      </div>

      <!-- Pagination -->
      <div v-if="media.total > media.perPage" class="pagination-wrapper mt-5">
        <b-pagination :total="media.total" :current.sync="media.page" :per-page="media.perPage"
          @change="onPageChange" />
      </div>
    </section>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
  },

  name: 'Media',

  props: {
    isModal: Boolean,
    type: { type: String, default: '' },
  },

  data() {
    return {
      form: {
        files: [],
      },
      toUpload: 0,
      uploaded: 0,
      showUploadForm: false,

      queryParams: {
        page: 1,
        query: '',
      },
    };
  },

  methods: {
    removeUploadFile(i) {
      this.form.files.splice(i, 1);
    },

    getMedia() {
      this.$api.getMedia({
        page: this.queryParams.page,
        query: this.queryParams.query,
      });
    },

    onToggleForm() {
      this.showUploadForm = !this.showUploadForm;
      this.$utils.setPref('media.upload', this.showUploadForm);
    },

    onQueryMedia() {
      this.queryParams.page = 1;
      this.getMedia();
    },

    onMediaSelect(m, e) {
      // If the component is open in the modal mode, close the modal and
      // fire the selection event.
      // Otherwise, do nothing and let the image open like a normal link.
      if (this.isModal) {
        e.preventDefault();
        this.$emit('selected', m);
        this.$parent.close();
      }
    },

    onSubmit() {
      this.toUpload = this.form.files.length;

      // Upload N files with N requests.
      for (let i = 0; i < this.toUpload; i += 1) {
        const params = new FormData();
        params.set('file', this.form.files[i]);
        this.$api.uploadMedia(params).then(() => {
          this.onUploaded();
        }, () => {
          this.onUploaded();
        });
      }
    },

    onDeleteMedia(id) {
      this.$api.deleteMedia(id).then(() => {
        this.getMedia();
      });
    },

    onUploaded() {
      this.uploaded += 1;
      if (this.uploaded >= this.toUpload) {
        this.toUpload = 0;
        this.uploaded = 0;
        this.form.files = [];

        this.getMedia();
      }
    },

    onPageChange(p) {
      this.queryParams.page = p;
      this.getMedia();
    },
  },

  computed: {
    ...mapState(['loading', 'media', 'serverConfig']),

    isProcessing() {
      if (this.toUpload > 0 && this.uploaded < this.toUpload) {
        return true;
      }
      return false;
    },
  },

  created() {
    this.$root.$on('page.refresh', this.getMedia);
  },

  destroyed() {
    this.$root.$off('page.refresh', this.getMedia);
  },

  mounted() {
    this.$api.getMedia();

    if (this.$utils.getPref('media.upload')) {
      this.showUploadForm = true;
    }
  },
});
</script>
