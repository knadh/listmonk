function Table({
  query = '',
  filters = {},
  total = 0,
  idSelector = 'input[name="id"]',
  toggleSelector = 'thead input[type="checkbox"]',
  ignoreQueryKeys = ['page', 'order', 'order_by'],
  valueParser = (value) => value,
  i18n,
} = {}) {
  return {
    query,
    filters,
    total,
    idSelector,
    toggleSelector,
    ignoreQueryKeys,
    valueParser,
    root: null,
    allSelected: false,
    selected: [],

    init(root) {
      this.root = root || this.$el || document;
      this._clearSelection();
    },

    onChange(event) {
      const el = event.target;
      if (!el.matches('input[type="checkbox"]')) {
        return;
      }

      if (el.matches(this.toggleSelector)) {
        this.onToggleVisible(event);
      } else if (el.matches(this.idSelector)) {
        this.onCheck();
      }
    },

    onCheck() {
      this.allSelected = false;
      this.selected = this._checkedValues();
      this._syncToggleState();
    },

    onToggleVisible(event) {
      this.root.querySelectorAll(this.idSelector).forEach((el) => {
        el.checked = event.target.checked;
      });
      this.onCheck();
    },

    onSelectAllQuery() {
      this.allSelected = true;
      this.selected = [];
      this.root.querySelectorAll(this.idSelector).forEach((el) => {
        el.checked = true;
      });
      this._syncToggleState();
    },

    _clearSelection() {
      this.root.querySelectorAll(this.idSelector).forEach((el) => {
        el.checked = false;
      });
      this.root.querySelectorAll(this.toggleSelector).forEach((el) => {
        el.checked = false;
      });
      this.allSelected = false;
      this.selected = [];
    },

    selectedQueryParams({ allKey = 'all', idKey = 'id', queryKey = 'query', } = {}) {
      const params = new URLSearchParams();
      if (this.allSelected) {
        params.set(queryKey, this.query || '');
        const ignoreKeys = new Set([...this.ignoreQueryKeys, allKey, idKey, queryKey]);
        Object.entries(this.filters).forEach(([key, values]) => {
          if (ignoreKeys.has(key)) {
            return;
          }
          (Array.isArray(values) ? values : [values]).forEach((value) => {
            if (value !== '' && value != null) {
              params.append(key, value);
            }
          });
        });
        params.set(allKey, 'true');
      } else {
        this.selected.forEach((id) => params.append(idKey, id));
      }

      return params;
    },

    get numSelected() {
      return this.allSelected ? this.total : this.selected.length;
    },

    get numSelectedLabel() {
      return i18n.ts('globals.messages.numSelected', { num: this.numSelected });
    },

    get selectAllLabel() {
      return i18n.ts('globals.messages.selectAll', { num: this.total });
    },

    _checkedValues() {
      return Array.from(this.root.querySelectorAll(`${this.idSelector}:checked`)).map((el) => this.valueParser(el.value));
    },

    _syncToggleState() {
      const visible = Array.from(this.root.querySelectorAll(this.idSelector));
      const checked = visible.filter((el) => el.checked).length;

      this.root.querySelectorAll(this.toggleSelector).forEach((el) => {
        el.checked = visible.length > 0 && checked === visible.length;
        el.indeterminate = checked > 0 && checked < visible.length;
      });
    },
  };
}

export function register(Alpine, i18n) {
  Alpine.data('table', (options = {}) => Table({ ...options, i18n }));
}
