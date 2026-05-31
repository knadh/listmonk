<template>
  <label class="oat-upload" @dragover.prevent @drop.prevent="onDrop">
    <input type="file" :multiple="multiple" :accept="accept || xaccept" :aria-label="label" @change="onChange">
    <slot />
  </label>
</template>

<script>
export default {
  name: 'OatUpload',
  props: {
    value: {
      type: [Array, File],
      default: null,
    },
    multiple: {
      type: Boolean,
      default: false,
    },
    accept: {
      type: String,
      default: '',
    },
    xaccept: {
      type: String,
      default: '',
    },
    label: {
      type: String,
      default: 'Upload file',
    },
  },
  methods: {
    emitFiles(files) {
      const list = Array.from(files);
      this.$emit('input', this.multiple ? list : list[0]);
    },
    onChange(e) {
      this.emitFiles(e.target.files);
    },
    onDrop(e) {
      this.emitFiles(e.dataTransfer.files);
    },
  },
};
</script>

<style>
.oat-upload {
  border: 1px dashed var(--border);
  border-radius: var(--radius);
  cursor: pointer;
  display: block;
  padding: var(--space-4);
}

.oat-upload input {
  display: none;
}
</style>
