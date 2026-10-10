import Alpine from 'alpinejs';
import {
  api,
  i18n,
} from '../main.js';
import * as u from '../utils.js';

function component() {
  return {
    // JSON meta details of a given bounce item viewed in the dialog.
    meta: {},
    metaTitle: '',

    onView(title, meta) {
      this.metaTitle = title;
      this.meta = meta || {};
      this.$refs.metaDialog.showModal();
    },

    async onDeleteBounce(id, email) {
      if (!(await u.confirm())) {
        return;
      }

      await api('bounces', `/bounces/${id}`, 'DELETE');
      u.reload({ message: i18n.ts('globals.messages.deleted', { name: email }) });
    },

    // ===============
    // Bulk actions.
    async onDeleteSelected(num, params) {
      if (num === 0 || !(await u.confirm(i18n.ts('globals.messages.confirmDelete', {
        num,
        name: i18n.tc('globals.terms.bounce', num).toLowerCase(),
      })))) {
        return;
      }

      await api('bounces', `/bounces?${params}`, 'DELETE');

      u.reload({
        message: i18n.ts('globals.messages.deletedCount', {
          num,
          name: i18n.tc('globals.terms.bounce', num),
        }),
      });
    },

    async onBlocklistSelected(allSelected, total, params) {
      const subIDs = [...new Set(Array.from(this.root.querySelectorAll('input[name="id"]:checked'))
        .map((el) => Number(el.dataset.subscriberId))
        .filter((id) => id > 0))];

      const num = allSelected ? total : subIDs.length;
      const message = i18n.ts('subscribers.confirmBlocklist', {
        num: allSelected ? i18n.t('globals.terms.all') : num,
      });
      if (num === 0 || !(await u.confirm(message))) {
        return;
      }

      await api('bounces', `/bounces/blocklist?${params}`, 'PUT');

      u.reload({ message: i18n.t('globals.messages.done') });
    },
  };
}

document.addEventListener('alpine:init', () => {
  Alpine.data('bouncesView', component);
}, { once: true });
