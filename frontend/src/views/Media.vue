<template>
  <section class="media-files">
    <h1 class="title is-4">
      {{ $t('media.title') }}
      <span v-if="media.length > 0">({{ media.length }})</span>

      <span class="has-text-grey-light"> / {{ settings['upload.provider'] }}</span>
    </h1>

    <b-loading :active="isProcessing || loading.media" />

    <section class="wrap">
      <form @submit.prevent="onSubmit" class="box">
        <div>
          <b-field :label="$t('media.uploadImage')">
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
    </section>

    <section class="wrap gallery mt-6">
      <b-table :data="media.results" :hoverable="true" :loading="loading.media" default-sort="createdAt" :paginated="true"
        backend-pagination pagination-position="both" @page-change="onPageChange" :current-page="media.page"
        :per-page="media.perPage" :total="media.total">
        <template #top-left>
          <div class="columns">
            <div class="column is-6">
              <form @submit.prevent="onQueryMedia">
                <div>
                  <b-field>
                    <b-input v-model="queryParams.query" name="query" expanded icon="magnify" ref="query"
                      data-cy="query" />
                    <p class="controls">
                      <b-button native-type="submit" type="is-primary" icon-left="magnify" data-cy="btn-query" />
                    </p>
                  </b-field>
                </div>
              </form>
            </div>
          </div>
        </template>

        <b-table-column v-slot="props" field="name" width="40%" :label="$t('globals.fields.name')">
          <a @click="(e) => onMediaSelect(props.row, e)" :href="props.row.url" target="_blank" rel="noopener noreferer"
            class="link" :title="props.row.filename">
            {{ props.row.filename }}
          </a>
        </b-table-column>

        <b-table-column v-slot="props" field="thumb" width="30%">
          <a @click="(e) => onMediaSelect(props.row, e)" :href="props.row.url" target="_blank" rel="noopener noreferer"
            class="thumb box">
            <img v-if="props.row.thumbUrl" :src="props.row.thumbUrl" :title="props.row.filename" alt="" />
            <span v-else class="ext">
              {{ props.row.filename.split(".").pop() }}
            </span>
          </a>
        </b-table-column>

        <b-table-column v-slot="props" field="created_at" width="25%" :label="$t('globals.fields.createdAt')" sortable>
          {{ $utils.niceDate(props.row.createdAt, true) }}
        </b-table-column>

        <b-table-column v-slot="props" field="actions" width="5%" cell-class="has-text-right">
          <a href="#" @click.prevent="$utils.confirm(null, () => onDeleteMedia(props.row.id))" data-cy="btn-delete"
            :aria-label="$t('globals.buttons.delete')">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </b-table-column>

        <template #empty v-if="!loading.media">
          <empty-placeholder />
        </template>
      </b-table>
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
    ...mapState(['loading', 'media', 'settings']),

    isProcessing() {
      if (this.toUpload > 0 && this.uploaded < this.toUpload) {
        return true;
      }
      return false;
    },
  },

  mounted() {
    this.$api.getMedia();
  },
});
</script>
