<template>
  <div class="items">
    <b-tabs :animated="false">
      <b-tab-item :label="$t('settings.appearance.adminName')" label-position="on-border">
        <div class="block">
          {{ $t('settings.appearance.adminHelp') }}
        </div>

        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border">
          <html-editor v-model="data['appearance.admin.custom_css']" name="body"
          language="css" />
        </b-field>

        <b-field :label="$t('settings.appearance.customJS')" label-position="on-border">
          <html-editor v-model="data['appearance.admin.custom_js']" name="body"
          language="css" />
        </b-field>
      </b-tab-item><!-- admin -->

      <b-tab-item :label="$t('settings.appearance.publicName')" label-position="on-border">
        <div class="block">
          {{ $t('settings.appearance.publicHelp') }}
        </div>

        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border">
          <html-editor v-model="data['appearance.public.custom_css']" name="body"
          language="css" />
        </b-field>

        <b-field :label="$t('settings.appearance.customJS')" label-position="on-border">
          <html-editor v-model="data['appearance.public.custom_js']" name="body"
          language="js" />
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
      type: Object,
    },
  },

  data() {
    return {
      data: this.form,
    };
  },

  computed: {
    ...mapState(['settings']),
  },
});

</script>
