<template>
  <div>
    <b-modal scroll="keep" @close="close" :aria-modal="true" :active="isVisible">
      <div>
        <div class="modal-card" style="width: auto">
          <header class="modal-card-head">
            <h4>{{ title }}</h4>
          </header>
        </div>
        <section expanded class="modal-card-body preview">
          <b-loading :active="isLoading" :is-full-page="false" />
          <form v-if="isPost" method="post" :action="previewURL" target="iframe" ref="form">
            <input v-if="templateId" type="hidden" name="template_id" :value="templateId" />
            <input v-if="contentType" type="hidden" name="content_type" :value="contentType" />
            <input v-if="templateType" type="hidden" name="template_type" :value="templateType" />
            <input v-if="archiveMeta" type="hidden" name="archive_meta" :value="archiveMeta" />
            <input v-if="body" type="hidden" name="body" :value="body" />
          </form>

          <iframe id="iframe" name="iframe" ref="iframe" :title="title" :src="isPost ? 'about:blank' : previewURL"
            @load="onLoaded" sandbox="allow-scripts" />
        </section>
        <footer class="modal-card-foot has-text-right">
          <b-button v-if="type === 'campaign'" icon-left="cloud-download-outline"
            :loading="isDownloading" :disabled="isDownloading" @click="downloadPDF">
            {{ $t('campaigns.downloadPDF') }}
          </b-button>
          <b-button @click="close">
            {{ $t('globals.buttons.close') }}
          </b-button>
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
    isPost: { type: Boolean, default: false },

    // Template or campaign ID.
    id: { type: Number, default: 0 },
    title: { type: String, default: '' },

    // campaign | template.
    type: { type: String, default: '' },

    // campaign | tx.
    templateType: { type: String, default: '' },

    archiveMeta: { type: String, default: null },

    body: { type: String, default: '' },
    contentType: { type: String, default: '' },
    templateId: { type: [Number, null], default: null },
    isArchive: { type: Boolean, default: false },
  },

  data() {
    return {
      isVisible: true,
      isLoading: true,
      formSubmitted: false,
      isDownloading: false,
    };
  },

  methods: {
    // Client-side "Save as PDF": render the campaign preview into a hidden iframe and open
    // the browser's print dialog (no backend or dependency). A dedicated frame keeps the
    // visible preview's sandbox intact and prints the full email; it is sandboxed
    // allow-same-origin + allow-modals (so the parent can call print()) without
    // allow-scripts. GET previews load previewURL; POST previews (editor/archive) re-submit
    // the existing preview form into the frame.
    downloadPDF() {
      if (this.isDownloading) {
        return;
      }
      this.isDownloading = true;

      // Browsers derive the print job's PDF title / suggested file name from the TOP
      // document's title (not the printed iframe's), so it's set below and restored on cleanup.
      const prevDocTitle = document.title;

      const frame = document.createElement('iframe');
      frame.style.display = 'none';
      frame.name = 'lm-pdf-frame';
      frame.sandbox = 'allow-same-origin allow-modals';

      // The fallback timer also cleans up if onload never fires or afterprint isn't emitted.
      let removed = false;
      let fallbackTimer = null;
      let printed = false;
      const cleanup = () => {
        if (removed) {
          return;
        }
        removed = true;
        clearTimeout(fallbackTimer);
        if (frame.parentNode) {
          frame.parentNode.removeChild(frame);
        }
        // Restore only if the title is still the one we set, so a title changed elsewhere
        // (e.g. by the router) while the print dialog was open isn't clobbered.
        if (document.title === this.title) {
          document.title = prevDocTitle;
        }
        this.isDownloading = false;
      };
      fallbackTimer = setTimeout(cleanup, 60000);

      frame.onload = () => {
        // A freshly created frame fires an initial about:blank load (and the POST form
        // submission adds another); wait for the real, non-empty content, then print once.
        const { contentDocument } = frame;
        if (!contentDocument || !contentDocument.body || contentDocument.body.children.length === 0) {
          return;
        }
        if (printed) {
          return;
        }
        printed = true;

        // Defer a tick before print(); a synchronous print() in onload can yield a blank page.
        setTimeout(() => {
          try {
            // Set the document title so the browser proposes the campaign name as the PDF
            // file name; restored in cleanup() once the print dialog closes.
            if (this.title) {
              document.title = this.title;
            }
            // Keep background colours/images in the print output (browsers drop them by
            // default). print-color-adjust is inherited, so setting it on <html> covers all.
            contentDocument.documentElement.style.setProperty('print-color-adjust', 'exact');
            contentDocument.documentElement.style.setProperty('-webkit-print-color-adjust', 'exact');
            frame.contentWindow.onafterprint = cleanup;
            frame.contentWindow.focus();
            frame.contentWindow.print();
          } catch (e) {
            this.$utils.toast(e.toString(), 'is-danger');
            cleanup();
          }
        }, 0);
      };

      document.body.appendChild(frame);

      if (this.isPost) {
        // Re-submit the existing preview form (body/template overrides) into the frame.
        const { form } = this.$refs;
        const prevTarget = form.target;
        form.target = 'lm-pdf-frame';
        form.submit();
        form.target = prevTarget;
      } else {
        frame.src = this.previewURL;
      }
    },

    close() {
      this.$emit('close');
      this.isVisible = false;
    },

    // On iframe load, kill the spinner.
    onLoaded() {
      if (!this.isPost) {
        this.isLoading = false;
        return;
      }

      if (this.formSubmitted) {
        this.isLoading = false;
      }
    },
  },

  computed: {
    previewURL() {
      let uri = 'about:blank';

      if (this.type === 'campaign') {
        uri = this.isArchive ? uris.previewCampaignArchive : uris.previewCampaign;
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
    if (this.isPost) {
      setTimeout(() => {
        this.$refs.form.submit();
        this.formSubmitted = true;
      }, 100);
    }
  },
};
</script>
