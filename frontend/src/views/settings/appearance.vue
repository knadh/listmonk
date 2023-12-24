<template>
  <div class="items">
    <b-tabs :animated="false" v-model="tab">
      <b-tab-item :label="$t('settings.appearance.adminName')" label-position="on-border">
        <div class="block">
          {{ $t('settings.appearance.adminHelp') }}
        </div>

        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border">
          <html-editor v-model="data['appearance.admin.custom_css']" name="body" language="css" />
        </b-field>

        <b-field :label="$t('settings.appearance.customJS')" label-position="on-border">
          <html-editor v-model="data['appearance.admin.custom_js']" name="body" language="css" />
        </b-field>
      </b-tab-item><!-- admin -->

      <b-tab-item :label="$t('settings.appearance.publicName')" label-position="on-border">
        <div class="block">
          {{ $t('settings.appearance.publicHelp') }}
        </div>

        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border">
          <html-editor v-model="data['appearance.public.custom_css']" name="body" language="css" />
        </b-field>

        <b-field :label="$t('settings.appearance.customJS')" label-position="on-border">
          <html-editor v-model="data['appearance.public.custom_js']" name="body" language="js" />
        </b-field>
      </b-tab-item><!-- public -->
    </b-tabs>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import HTMLEditor from '../../components/HTMLEditor.vue';

export default Vue.extend({
  components: {
    'html-editor': HTMLEditor,
  },

  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
      tab: 0,
    };
  },

  mounted() {
    this.tab = this.$utils.getPref('settings.apperanceTab') || 0;
  },

  watch: {
    tab(t) {
      this.$utils.setPref('settings.apperanceTab', t);
    },
  },

  computed: {
    ...mapState(['settings']),
  },
});

</script>
