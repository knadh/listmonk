<template>
  <input
    :type="datetime ? 'datetime-local' : 'date'"
    :name="name"
    :required="required"
    :aria-label="name || 'date'"
    :value="formatted"
    @input="onInput"
  >
</template>

<script>
function pad(v) {
  return `${v}`.padStart(2, '0');
}

export default {
  name: 'OatDateInput',
  props: {
    value: {
      type: [Date, String],
      default: null,
    },
    datetime: {
      type: Boolean,
      default: false,
    },
    name: {
      type: String,
      default: '',
    },
    required: {
      type: Boolean,
      default: false,
    },
  },
  computed: {
    formatted() {
      if (!this.value) {
        return '';
      }
      const d = this.value instanceof Date ? this.value : new Date(this.value);
      const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
      if (!this.datetime) {
        return date;
      }
      return `${date}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
    },
  },
  methods: {
    onInput(e) {
      this.$emit('input', e.target.value ? new Date(e.target.value) : null);
    },
  },
};
</script>
