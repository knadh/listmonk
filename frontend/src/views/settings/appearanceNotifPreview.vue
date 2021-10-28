<template>
  <div>
    <b-modal scroll="keep" @close="close" :aria-modal="true" :active="isVisible">
      <div>
        <div class="modal-card" style="width: auto">
          <header class="modal-card-head">
            <h4>{{ previewTitle }}</h4>
          </header>
        </div>
        <section expanded class="modal-card-body preview">
          <b-loading :active="isLoading" :is-full-page="false"></b-loading>
          <!-- eslint-disable-next-line max-len -->
          <iframe id="iframe" name="iframe" ref="iframe" :title="previewTitle" :src="previewURL"></iframe>
        </section>
        <footer class="modal-card-foot has-text-right">
          <b-button @click="close">{{ $t('globals.buttons.close') }}</b-button>
        </footer>
      </div>
    </b-modal>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  props: {
    previewTitle: String,
  },

  data() {
    return {
      isVisible: true,
      isLoading: true,

      // preview is dynamically generated and returned from http handler
      previewURL: '/api/admin/template/preview',
    };
  },

  methods: {
    close() {
      this.$emit('close');
      this.isVisible = false;
    },
  },

  mounted() {
    this.isLoading = false;
  },
});

</script>
