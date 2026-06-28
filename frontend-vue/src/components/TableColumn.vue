<script>
export default {
  name: 'BTableColumn',
  inject: ['bTable'],
  props: {
    field: {
      type: String,
      default: '',
    },
    label: {
      type: String,
      default: '',
    },
    sortable: {
      type: Boolean,
      default: false,
    },
    numeric: {
      type: Boolean,
      default: false,
    },
    centered: {
      type: Boolean,
      default: false,
    },
    align: {
      type: String,
      default: '',
    },
    cellClass: {
      type: String,
      default: '',
    },
    headerClass: {
      type: String,
      default: '',
    },
    width: {
      type: [String, Number],
      default: '',
    },
    tdAttrs: {
      type: Function,
      default: null,
    },
  },
  created() {
    this.bTable.registerColumn(this);
  },
  beforeDestroy() {
    this.bTable.unregisterColumn(this);
  },
  render() {
    return null;
  },
};
</script>
