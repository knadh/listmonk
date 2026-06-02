<template>
  <div data-field>
    <label v-if="label" :for="labelFor">{{ label }}</label>
    <slot />
    <small v-if="message" data-hint>{{ message }}</small>
  </div>
</template>

<script>
export default {
  name: 'BField',
  data() {
    return {
      generatedID: '',
      fieldID: `b-field-${Date.now()}-${Math.floor(Math.random() * 1000).toString().padStart(3, '0')}`,
    };
  },
  props: {
    label: {
      type: String,
      default: '',
    },
    for: {
      type: String,
      default: '',
    },
    message: {
      type: String,
      default: '',
    },
  },
  computed: {
    labelFor() {
      return this.for || this.generatedID;
    },
  },
  mounted() {
    this.setGeneratedID();
  },
  updated() {
    this.setGeneratedID();
  },
  methods: {
    setGeneratedID() {
      if (!this.label || this.for) {
        return;
      }

      const control = this.$el.querySelector('input:not([type="hidden"]), select, textarea');
      if (!control) {
        return;
      }

      if (!control.id) {
        control.id = this.fieldID;
      }
      this.generatedID = control.id;
    },
  },
};
</script>
