<template>
  <dialog ref="dialog" class="b-modal" :style="contentStyle" @close="onNativeClose" @cancel="onCancel">
    <slot />
  </dialog>
</template>

<script>
export default {
  name: 'BModal',
  props: {
    active: {
      type: Boolean,
      default: false,
    },
    width: {
      type: [Number, String],
      default: '',
    },
    canCancel: {
      type: [Boolean, Array],
      default: true,
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
      if (this.canCancel === false) {
        return;
      }
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
.b-modal {
  max-width: min(96vw, 1200px);
  width: min(100% - 2rem, 32rem);
}

.b-modal>form>.dialog-card,
.b-modal>.dialog-card {
  display: contents;
}

.b-modal>form>.dialog-card>header,
.b-modal>.dialog-card>header {
  background: var(--faint);
  padding: var(--space-6);
  padding-block-end: var(--space-2);
  margin-block-end: var(--space-4);
}

.b-modal>form>.dialog-card> :is(div, section),
.b-modal>.dialog-card> :is(div, section) {
  max-height: 70vh;
  overflow-y: auto;
  padding: var(--space-6);
  padding-block-start: 0;
}

.b-modal>form>.dialog-card>footer,
.b-modal>.dialog-card>footer {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
  padding: var(--space-6);
  padding-block-start: 0;
}
</style>
