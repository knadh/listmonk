<template>
  <div>
    <b-modal scroll="keep" @close="close"
      :aria-modal="true" :active="isVisible">
      <div>
        <div class="modal-card" style="width: auto">
          <header class="modal-card-head">
            <h4>{{ title }}</h4>
          </header>
        </div>
        <section expanded class="modal-card-body preview">
          <b-loading :active="isLoading" :is-full-page="false"></b-loading>
          <form v-if="body" method="post" :action="previewURL" target="iframe" ref="form">
            <input type="hidden" name="template_id" :value="templateId" />
            <input type="hidden" name="content_type" :value="contentType" />
            <input type="hidden" name="template_type" :value="templateType" />
            <input type="hidden" name="body" :value="body" />
          </form>

          <iframe id="iframe" name="iframe" ref="iframe"
            :title="title"
            :src="body ? 'about:blank' : previewURL"
            @load="onLoaded"
          ></iframe>
        </section>
        <footer class="modal-card-foot has-text-right">
          <b-button @click="close">{{ $t('globals.buttons.close') }}</b-button>
        </footer>
      </div>
    </b-modal>
  </div>
</template>

<script>
import { uris } from '../constants';

export default {
  name: 'CampaignPreview',

  props: {
    // Template or campaign ID.
    id: Number,
    title: String,

    // campaign | template.
    type: String,

    // campaign | tx.
    templateType: String,

    body: String,
    contentType: String,
    templateId: {
      type: Number,
      default: 0,
    },
  },

  data() {
    return {
      isVisible: true,
      isLoading: true,
    };
  },

  methods: {
    close() {
      this.$emit('close');
      this.isVisible = false;
    },

    // On iframe load, kill the spinner.
    onLoaded(l) {
      if (l.srcElement.contentWindow.location.href === 'about:blank') {
        return;
      }
      this.isLoading = false;
    },
  },

  computed: {
    previewURL() {
      let uri = 'about:blank';

      if (this.type === 'campaign') {
        uri = uris.previewCampaign;
      } else if (this.type === 'template') {
        if (this.id) {
          uri = uris.previewTemplate;
        } else {
          uri = uris.previewRawTemplate;
        }
      }

      return uri.replace(':id', this.id);
    },
  },

  mounted() {
    setTimeout(() => {
      if (this.$refs.form) {
        this.$refs.form.submit();
      }
    }, 100);
  },
};
</script>
