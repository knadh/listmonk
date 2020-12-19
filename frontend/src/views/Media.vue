<template>
  <section class="media-files">
    <h1 class="title is-4">{{ $t('media.title') }}
      <span v-if="media.length > 0">({{ media.length }})</span>

      <span class="has-text-grey-light"> / {{ serverConfig.mediaProvider }}</span>
    </h1>

    <b-loading :active="isProcessing || loading.media"></b-loading>

    <section class="wrap-small">
      <form @submit.prevent="onSubmit" class="box">
        <div>
          <b-field :label="$t('media.uploadImage')">
            <b-upload
              v-model="form.files"
              drag-drop
              multiple
              accept=".png,.jpg,.jpeg,.gif"
              expanded>
              <div class="has-text-centered section">
                <p>
                  <b-icon icon="file-upload-outline" size="is-large"></b-icon>
                </p>
                <p>{{ $t('media.uploadHelp') }}</p>
              </div>
            </b-upload>
          </b-field>
          <div class="tags" v-if="form.files.length > 0">
            <b-tag v-for="(f, i) in form.files" :key="i" size="is-medium"
              closable @close="removeUploadFile(i)">
              {{ f.name }}
            </b-tag>
          </div>
          <div class="buttons">
            <b-button native-type="submit" type="is-primary" icon-left="file-upload-outline"
              :disabled="form.files.length === 0"
              :loading="isProcessing">{{ $tc('media.upload') }}</b-button>
          </div>
        </div>
      </form>
    </section>

    <section class="section gallery">
      <div v-for="group in items" :key="group.title">
        <h3 class="title is-5">{{ group.title }}</h3>

        <div class="thumbs">
          <div v-for="m in group.items" :key="m.id" class="box thumb">
            <a @click="(e) => onMediaSelect(m, e)" :href="m.url" target="_blank">
              <img :src="m.thumbUrl" :title="m.filename" />
            </a>
            <span class="caption is-size-7" :title="m.filename">{{ m.filename }}</span>

            <div class="actions has-text-right">
              <a :href="m.url" target="_blank">
                  <b-icon icon="arrow-top-right" size="is-small" />
              </a>
              <a href="#" @click.prevent="$utils.confirm(null, () => deleteMedia(m.id))">
                  <b-icon icon="trash-can-outline" size="is-small" />
              </a>
            </div>
          </div>
        </div>
        <hr />
      </div>
    </section>

  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import dayjs from 'dayjs';

export default Vue.extend({
  name: 'Media',

  props: {
    isModal: Boolean,
  },

  data() {
    return {
      form: {
        files: [],
      },
      toUpload: 0,
      uploaded: 0,
    };
  },

  methods: {
    removeUploadFile(i) {
      this.form.files.splice(i, 1);
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

    deleteMedia(id) {
      this.$api.deleteMedia(id).then(() => {
        this.$api.getMedia();
      });
    },

    onUploaded() {
      this.uploaded += 1;
      if (this.uploaded >= this.toUpload) {
        this.toUpload = 0;
        this.uploaded = 0;
        this.form.files = [];

        this.$api.getMedia();
      }
    },
  },

  computed: {
    ...mapState(['media', 'serverConfig', 'loading']),

    isProcessing() {
      if (this.toUpload > 0 && this.uploaded < this.toUpload) {
        return true;
      }
      return false;
    },

    // Filters the list of media items by months into:
    // [{"title": "Jan 2020", items: [...]}, ...]
    items() {
      const out = [];
      if (!this.media || !(this.media instanceof Array)) {
        return out;
      }

      let lastStamp = '';
      let lastIndex = 0;
      this.media.forEach((m) => {
        const stamp = dayjs(m.createdAt).format('MMM YYYY');
        if (stamp !== lastStamp) {
          out.push({ title: stamp, items: [] });
          lastStamp = stamp;
          lastIndex = out.length;
        }

        out[lastIndex - 1].items.push(m);
      });
      return out;
    },
  },

  mounted() {
    this.$api.getMedia();
  },
});
</script>
