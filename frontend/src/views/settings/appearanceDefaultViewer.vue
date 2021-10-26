<template>
  <section>
    <div class="content template-modal-content">
      <header class="modal-card-head">
        <h4> {{ name }}</h4>
      </header>
      <section expanded class="modal-card-body">
        <b-field :label="$t('templates.rawHTML')" label-position="on-border">
          <html-editor v-model="body" :disabled="disabled" />
        </b-field>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">{{ $t('globals.buttons.close') }}</b-button>
      </footer>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import HTMLEditor from '../../components/HTMLEditor.vue';

export default Vue.extend({
  components: {
    'html-editor': HTMLEditor,
  },

  props: {
    defaultName: String,
    defaultBody: String,
  },

  data() {
    return {
      disabled: true,
      name: this.defaultName,
      body: this.defaultBody,
    };
  },
});

</script>
