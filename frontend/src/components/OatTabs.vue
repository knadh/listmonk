<script>
export default {
  name: 'OatTabs',
  provide() {
    return {
      oatTabs: this,
    };
  },
  props: {
    value: {
      type: [String, Number],
      default: 0,
    },
  },
  data() {
    return {
      tabs: [],
      active: this.value,
    };
  },
  watch: {
    value(v) {
      this.active = v;
    },
  },
  methods: {
    registerTab(tab) {
      if (!this.tabs.includes(tab)) {
        this.tabs.push(tab);
        if (this.tabs.length === 1 && (this.active === null || this.active === undefined || this.active === 0)) {
          this.active = tab.tabValue;
        }
        this.$forceUpdate();
      }
    },
    unregisterTab(tab) {
      this.tabs = this.tabs.filter((t) => t !== tab);
    },
    setActive(tab) {
      if (tab.disabled) {
        return;
      }
      this.active = tab.tabValue;
      this.$emit('input', this.active);
    },
  },
  render(h) {
    return h('div', { class: 'oat-tabs' }, [
      h('div', { attrs: { role: 'tablist' } }, this.tabs.map((tab) => h('button', {
        attrs: { type: 'button', role: 'tab', disabled: tab.disabled },
        class: { outline: this.active !== tab.tabValue },
        on: { click: () => this.setActive(tab) },
      }, tab.label))),
      h('div', this.tabs.map((tab) => h('app-section', {
        directives: [{ name: 'show', value: this.active === tab.tabValue }],
        attrs: { role: 'tabpanel' },
      }, tab.$slots.default))),
      h('div', { style: { display: 'none' } }, this.$slots.default),
    ]);
  },
};
</script>

<style>
.oat-tabs > [role="tablist"] {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-block-end: var(--space-4);
}
</style>
