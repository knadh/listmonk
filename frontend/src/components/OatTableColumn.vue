<script>
export default {
  name: 'OatTableColumn',
  inject: ['oatTable'],
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
    this.oatTable.registerColumn(this);
  },
  beforeDestroy() {
    this.oatTable.unregisterColumn(this);
  },
  render() {
    return null;
  },
};
</script>
