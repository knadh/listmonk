<template>
  <nav v-if="numPages > 1" class="oat-pagination" :aria-label="$t ? $t('globals.terms.page') : 'Pagination'">
    <button type="button" :disabled="page <= 1" @click="setPage(page - 1)">
      <oat-icon icon="chevron-left" />
    </button>
    <button v-for="p in pages" :key="p" type="button" :class="{ outline: p !== page }"
      :aria-current="p === page ? 'page' : null" @click="setPage(p)">
      {{ p }}
    </button>
    <button type="button" :disabled="page >= numPages" @click="setPage(page + 1)">
      <oat-icon icon="chevron-right" />
    </button>
  </nav>
</template>

<script>
export default {
  name: 'OatPagination',
  props: {
    total: {
      type: Number,
      default: 0,
    },
    current: {
      type: Number,
      default: 1,
    },
    perPage: {
      type: Number,
      default: 20,
    },
  },
  computed: {
    page() {
      return this.current || 1;
    },
    numPages() {
      return Math.max(1, Math.ceil(this.total / this.perPage));
    },
    pages() {
      const out = [];
      const start = Math.max(1, this.page - 2);
      const end = Math.min(this.numPages, start + 4);
      for (let i = start; i <= end; i += 1) {
        out.push(i);
      }
      return out;
    },
  },
  methods: {
    setPage(page) {
      if (page < 1 || page > this.numPages || page === this.page) {
        return;
      }
      this.$emit('update:current', page);
      this.$emit('change', page);
      this.$emit('page-change', page);
    },
  },
};
</script>

<style>
.oat-pagination {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  justify-content: flex-end;
  margin-block: var(--space-4);
}
</style>
