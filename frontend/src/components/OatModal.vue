<template>
  <dialog ref="dialog" class="oat-modal" :style="contentStyle" @close="onNativeClose" @cancel="onCancel">
    <slot />
  </dialog>
</template>

<script>
export default {
  name: 'OatModal',
  props: {
    active: {
      type: Boolean,
      default: false,
    },
    width: {
      type: [Number, String],
      default: '',
    },
  },
  computed: {
    contentStyle() {
      if (!this.width) {
        return {};
      }
      return { width: typeof this.width === 'number' ? `${this.width}px` : this.width };
    },
  },
  watch: {
    active: {
      immediate: true,
      handler(active) {
        this.$nextTick(() => {
          const { dialog } = this.$refs;
          if (!dialog) {
            return;
          }
          if (active && !dialog.open) {
            dialog.showModal();
          } else if (!active && dialog.open) {
            dialog.close();
          }
        });
      },
    },
  },
  methods: {
    close() {
      this.$emit('update:active', false);
      this.$emit('close');
    },
    onCancel(e) {
      e.preventDefault();
      this.close();
    },
    onNativeClose() {
      if (this.active) {
        this.close();
      }
    },
  },
};
</script>

<style>
.oat-modal {
  max-width: min(96vw, 1200px);
  width: min(100% - 2rem, 32rem);
}

.oat-modal>form>.dialog-card,
.oat-modal>.dialog-card {
  display: contents;
}

.oat-modal>form>.dialog-card>header,
.oat-modal>.dialog-card>header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-6);
  padding-block-end: 0;
}

.oat-modal>form>.dialog-card> :is(div, section),
.oat-modal>.dialog-card> :is(div, section) {
  max-height: 70vh;
  overflow-y: auto;
  padding: var(--space-6);
}

.oat-modal>form>.dialog-card>footer,
.oat-modal>.dialog-card>footer {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
  padding: var(--space-6);
  padding-block-start: 0;
}
</style>
