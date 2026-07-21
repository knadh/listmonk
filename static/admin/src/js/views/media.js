import {
  urls,
  api,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

function component() {
  return {
    files: [],
    isUploading: false,

    // ===============
    // Event handlers.
    onOpenUpload() {
      this.files = [];
      this.$refs.dialog.showModal();
    },

    onClose() {
      this.$refs.dialog.close();
    },

    async onUpload() {
      if (this.files.length === 0) {
        return;
      }

      this.isUploading = true;

      // Upload each file with an independent request.
      const results = await Promise.all(this.files.map((file) => {
        const params = new FormData();
        params.set('file', file);

        return fetch(`${urls.api}/media`, {
          method: 'POST',
          body: params,
          headers: { Accept: 'application/json' },
        }).then(async (resp) => {
          if (!resp.ok) {
            const out = await resp.json().catch(() => ({}));
            throw new Error(out.message || resp.statusText);
          }
        }).catch((err) => {
          u.toast(`${file.name}: ${err.message}`, 'danger');
        });
      }));

      this.isUploading = false;
      u.reload({ message: i18n.ts('globals.messages.done') });
      return results;
    },

    async onDelete(id, name) {
      if (!(await u.confirm())) {
        return;
      }

      await api('media', `/media/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name }) });
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('mediaView', component);
}, { once: true });
