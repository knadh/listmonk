<template>
  <section class="media-files">
    <header v-if="!isModal" class="row page-header">
      <div class="col-8">
        <h1>
          {{ $t('media.title') }}
          <span v-if="media.results && media.results.length > 0">({{ media.results.length }})</span>
          <span class="text-lighter text-7"> / {{ serverConfig.media_provider }}</span>
        </h1>
      </div>
    </header>

    <div :class="isModal ? 'media-content' : 'card page-content'">
      <oat-loading :active="isProcessing || loading.media" />

      <section class="gallery">
        <div class="row mb-4">
          <div class="col-12">
            <form @submit.prevent="onQueryMedia" class="search">
              <fieldset class="group">
                <input aria-label="Search" v-model="queryParams.query" name="query" ref="query" data-cy="query"
                  placeholder="Search">
                <button type="submit" data-variant="primary" data-cy="btn-query" aria-label="Search">
                  <oat-icon icon="magnify" />
                </button>
              </fieldset>
            </form>
          </div>
          <div v-if="$can('media:manage')" class="col-2">
            <button type="button" @click="onToggleForm" data-cy="btn-toggle-upload">
              {{ $t('media.upload') }}
            </button>
          </div>
        </div>

        <div>
          <form @submit.prevent="onSubmit" class="mb-6" data-cy="upload">
            <div>
              <oat-field :label="$t('media.upload')">
                <oat-upload v-model="form.files" multiple xaccept=".png,.jpg,.jpeg,.gif,.svg">
                  <div class="align-center app-section">
                    <p>
                      <oat-icon icon="file-upload-outline" />
                    </p>
                    <p>{{ $t('media.uploadHelp') }}</p>
                  </div>
                </oat-upload>
              </oat-field>
              <div class="hstack" v-if="form.files.length > 0">
                <span v-for="(f, i) in form.files" :key="i" closable @close="removeUploadFile(i)">
                  {{ f.name }}
                </span>
              </div>
              <div class="hstack">
                <button type="submit" data-variant="primary" :disabled="form.files.length === 0"
                  :loading="isProcessing">
                  {{ $tc('media.upload') }}
                </button>
              </div>
            </div>
          </form>
        </div>

        <!-- Pagination -->
        <div v-if="media.total > media.perPage" class="pagination-wrapper mt-5">
          <oat-pagination :total="media.total" :current.sync="media.page" :per-page="media.perPage"
            @change="onPageChange" />
        </div>

        <div v-if="loading.media" class="align-center py-6">
          <oat-loading :active="loading.media" />
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
                  <oat-icon icon="trash-can-outline" />
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
          <oat-pagination :total="media.total" :current.sync="media.page" :per-page="media.perPage"
            @change="onPageChange" />
        </div>
      </section>
    </div>
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
