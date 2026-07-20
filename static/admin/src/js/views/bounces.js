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
    async onDeleteSelected(allSelected, selected, total) {
      const num = allSelected ? total : selected.length;
      if (num === 0 || !(await u.confirm(i18n.ts('globals.messages.confirmDelete', {
        num,
        name: i18n.tc('globals.terms.bounce', num).toLowerCase(),
      })))) {
        return;
      }

      if (allSelected) {
        await api('bounces', '/bounces?all=true', 'DELETE');
      } else {
        const q = new URLSearchParams();
        selected.forEach((id) => q.append('id', id));
        await api('bounces', `/bounces?${q.toString()}`, 'DELETE');
      }

      u.reload({
        message: i18n.ts('globals.messages.deletedCount', {
          num,
          name: i18n.tc('globals.terms.bounce', num),
        }),
      });
    },

    async onBlocklistSelected(allSelected, total) {
      const subIDs = Array.from(document.querySelectorAll('input[name="id"]:checked'))
        .map((el) => Number(el.dataset.subscriberId))
        .filter((id) => id > 0);

      const num = allSelected ? total : subIDs.length;
      if (num === 0 || !(await u.confirm(i18n.ts('subscribers.confirmBlocklist', { num })))) {
        return;
      }

      // "Select all" blocklists every bounced subscriber or blocklist selected rows.
      if (allSelected) {
        await api('bounces', '/bounces/blocklist', 'PUT');
      } else {
        await api('bounces', '/subscribers/blocklist', 'PUT', { ids: subIDs });
      }

      u.reload({ message: i18n.t('globals.messages.done') });
    },
  };
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('bouncesView', component);
}, { once: true });
