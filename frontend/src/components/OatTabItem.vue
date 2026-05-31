<script>
export default {
  name: 'OatTabItem',
  inject: ['oatTabs'],
  props: {
    label: {
      type: String,
      default: '',
    },
    value: {
      type: [String, Number],
      default: null,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  computed: {
    tabValue() {
      return this.value === null || this.value === undefined ? this.label : this.value;
    },
  },
  created() {
    this.oatTabs.registerTab(this);
  },
  beforeDestroy() {
    this.oatTabs.unregisterTab(this);
  },
  render() {
    return null;
  },
};
</script>
