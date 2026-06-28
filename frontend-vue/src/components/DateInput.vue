<template>
  <input v-bind="$attrs" :type="datetime ? 'datetime-local' : 'date'" :name="name" :required="required"
    :disabled="disabled" :aria-label="placeholder || name || 'date'" :placeholder="placeholder" :value="formatted"
    @input="onInput">
</template>

<script>
function pad(v) {
  return `${v}`.padStart(2, '0');
}

export default {
  name: 'BDatepicker',
  inheritAttrs: false,
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
    disabled: {
      type: Boolean,
      default: false,
    },
    placeholder: {
      type: String,
      default: '',
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
