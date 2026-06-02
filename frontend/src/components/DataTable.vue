<script>
export default {
  name: 'BTable',
  provide() {
    return {
      bTable: this,
    };
  },
  props: {
    data: {
      type: Array,
      default: () => [],
    },
    loading: {
      type: Boolean,
      default: false,
    },
    checkable: {
      type: Boolean,
      default: false,
    },
    checkedRows: {
      type: Array,
      default: () => [],
    },
    paginated: {
      type: Boolean,
      default: false,
    },
    perPage: {
      type: Number,
      default: 20,
    },
    total: {
      type: Number,
      default: 0,
    },
    currentPage: {
      type: Number,
      default: 1,
    },
    backendPagination: {
      type: Boolean,
      default: false,
    },
    backendSorting: {
      type: Boolean,
      default: false,
    },
    defaultSort: {
      type: String,
      default: '',
    },
    defaultSortDirection: {
      type: String,
      default: 'asc',
    },
    detailed: {
      type: Boolean,
      default: false,
    },
    rowClass: {
      type: Function,
      default: null,
    },
  },
  data() {
    return {
      columns: [],
      sortField: this.defaultSort,
      sortDirection: this.defaultSortDirection,
      localPage: this.currentPage || 1,
    };
  },
  computed: {
    page() {
      return this.currentPage || this.localPage || 1;
    },
    totalRows() {
      return this.total || this.sortedRows.length;
    },
    sortedRows() {
      const rows = [...this.data];
      if (!this.sortField || this.backendSorting) {
        return rows;
      }
      const dir = this.sortDirection === 'desc' ? -1 : 1;
      return rows.sort((a, b) => {
        if (a[this.sortField] === b[this.sortField]) {
          return 0;
        }
        return a[this.sortField] > b[this.sortField] ? dir : -dir;
      });
    },
    visibleRows() {
      if (!this.paginated || this.backendPagination) {
        return this.sortedRows;
      }
      const start = (this.page - 1) * this.perPage;
      return this.sortedRows.slice(start, start + this.perPage);
    },
    allChecked() {
      return this.visibleRows.length > 0 && this.visibleRows.every((row) => this.checkedRows.includes(row));
    },
  },
  methods: {
    registerColumn(column) {
      if (!this.columns.includes(column)) {
        this.columns.push(column);
        this.$forceUpdate();
      }
    },
    unregisterColumn(column) {
      this.columns = this.columns.filter((c) => c !== column);
    },
    cellValue(row, col) {
      return col.field ? row[col.field] : '';
    },
    sort(col) {
      if (!col.sortable || !col.field) {
        return;
      }
      if (this.sortField === col.field) {
        this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
      } else {
        this.sortField = col.field;
        this.sortDirection = 'asc';
      }
      this.$emit('sort', this.sortField, this.sortDirection);
    },
    setPage(page) {
      this.localPage = page;
      this.$emit('page-change', page);
    },
    emitChecked(rows, eventName) {
      this.$emit('update:checkedRows', rows);
      this.$emit(eventName, rows);
    },
    toggleRow(row) {
      const rows = this.checkedRows.includes(row)
        ? this.checkedRows.filter((r) => r !== row)
        : [...this.checkedRows, row];
      this.emitChecked(rows, 'check');
    },
    toggleAll(e) {
      const rows = e.target.checked
        ? [...new Set([...this.checkedRows, ...this.visibleRows])]
        : this.checkedRows.filter((row) => !this.visibleRows.includes(row));
      this.emitChecked(rows, 'check-all');
    },
  },
  render(h) {
    const hiddenColumns = h('div', { style: { display: 'none' } }, this.$slots.default);
    const headerCells = this.columns.map((col) => h('th', {
      class: col.headerClass,
      style: col.width ? { width: typeof col.width === 'number' ? `${col.width}px` : col.width } : {},
    }, [
      col.sortable
        ? h('button', { attrs: { type: 'button' }, class: 'ghost', on: { click: () => this.sort(col) } }, col.label)
        : col.label,
    ]));

    if (this.checkable) {
      headerCells.unshift(h('th', [
        h('input', {
          attrs: { type: 'checkbox', 'aria-label': 'Select all' },
          domProps: { checked: this.allChecked },
          on: { change: this.toggleAll },
        }),
      ]));
    }

    const bodyRows = [];
    this.visibleRows.forEach((row, index) => {
      const cells = this.columns.map((col) => {
        const slot = col.$scopedSlots.default;
        const content = slot ? slot({ row, index }) : this.cellValue(row, col);
        const attrs = col.tdAttrs ? col.tdAttrs(row) : {};
        return h('td', {
          class: [
            col.cellClass,
            { numeric: col.numeric, 'align-center': col.centered, 'align-right': col.align === 'right' },
          ],
          attrs,
        }, content);
      });

      if (this.checkable) {
        cells.unshift(h('td', [
          h('input', {
            attrs: { type: 'checkbox', 'aria-label': 'Select row' },
            domProps: { checked: this.checkedRows.includes(row) },
            on: { change: () => this.toggleRow(row) },
          }),
        ]));
      }

      bodyRows.push(h('tr', { class: this.rowClass ? this.rowClass(row, index) : '' }, cells));
      if (this.detailed && this.$scopedSlots.detail) {
        bodyRows.push(h('tr', { class: 'details' }, [
          h('td', { attrs: { colspan: this.columns.length + (this.checkable ? 1 : 0) } }, this.$scopedSlots.detail({ row, index })),
        ]));
      }
    });

    const topLeft = this.$slots['top-left'];
    const topRight = this.$slots['top-right'];
    const toolbar = (topLeft || topRight) ? h('div', { class: 'table-toolbar' }, [
      h('div', { class: 'table-toolbar-left' }, topLeft),
      h('div', { class: 'table-toolbar-right' }, topRight),
    ]) : null;

    return h('div', {
      class: 'b-table',
      attrs: {
        'aria-busy': this.loading ? 'true' : null,
        'data-spinner': 'large overlay',
      },
    }, [
      hiddenColumns,
      toolbar,
      h('div', { class: 'table' }, [h('table', [
        h('thead', [h('tr', headerCells)]),
        h('tbody', bodyRows),
      ])]),
      this.paginated ? h('b-pagination', {
        props: { total: this.totalRows, current: this.page, perPage: this.perPage },
        on: { change: this.setPage },
      }) : null,
    ]);
  },
};
</script>

<style>
.b-table {
  overflow-x: auto;
  position: relative;
}

.table-toolbar {
  align-items: flex-start;
  display: flex;
  gap: var(--space-4);
  justify-content: space-between;
  margin-block-end: var(--space-4);
}

.table-toolbar-left {
  flex: 1 1 auto;
  min-width: 0;
}

.table-toolbar-right {
  flex: 0 0 auto;
}

.b-table table {
  width: 100%;
}

.b-table .actions {
  white-space: nowrap;
}

.b-table tr.details td {
  background: var(--muted);
}
</style>
