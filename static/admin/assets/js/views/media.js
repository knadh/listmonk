import Alpine from 'alpinejs';
import { autocomp } from '@knadh/autocomp';
import { tagAutocomplete } from '../tag-selector.js';
import {
  urls,
  api,
  i18n,
  isLoading,
} from '../main.js';
import * as u from '../utils.js';

function component(mode, tags) {
  const tagNames = tags.map((t) => t.tag);
  return {
    normalizeTags: u.normalizeTags,

    files: [],
    isLoading,
    tags: [],
    media: null,
    editTags: [],
    isUploading: false,
    isPicker: mode === 'picker',

    tagURL(tag) {
      const url = new URL(window.location.href);
      url.searchParams.delete('page');
      url.searchParams.set('tag', tag);
      return url.href;
    },

    fileExt(filename) {
      const i = filename.lastIndexOf('.');
      return i > 0 ? filename.slice(i + 1).toUpperCase() : '';
    },

    get hasTagChanges() {
      return this.media && JSON.stringify(u.normalizeTags(this.editTags).sort()) !== JSON.stringify(u.normalizeTags(this.media.tags).sort());
    },

    autocomplete(el, upload = false) {
      const query = (q) => tagNames.filter((tag) => tag.toLowerCase().includes(q.toLowerCase()));
      if (upload) {
        tagAutocomplete(el, { query, render: (tag, selected) => (selected ? tag : `#${tag}`) });
      } else {
        autocomp(el, {
          onQuery: (q) => query(q).slice(0, 10),
          onRender: (tag) => document.createTextNode(`#${tag}`),
          onSelect: (tag) => tag,
        });
      }
    },


    // ===============
    // Event handlers.
    onPreview(m) {
      this.media = m;
      this.editTags = [...m.tags];
      this.$nextTick(() => this.$refs.previewDialog.showModal());
    },

    // Post the picked media to the parent window (picker runs inside an iframe).
    onPick(m) {
      window.parent.postMessage({ type: 'media-select', media: Alpine.raw(m) }, window.location.origin);
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
        u.normalizeTags(this.tags).forEach((tag) => params.append('tags', tag));

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

    async onSaveTags() {
      if (!this.hasTagChanges || isLoading('media.save')) return;

      await api('media.save', `/media/${this.media.id}`, 'PUT', {
        tags: u.normalizeTags(this.editTags),
      });
      u.reload({ message: i18n.ts('globals.messages.updated', { name: this.media.filename }) });
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
  Alpine.data('mediaView', (mode, tags) => component(mode, tags));
}, { once: true });
