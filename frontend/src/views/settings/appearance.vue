<template>
  <div class="items">
    <b-tabs :animated="false" v-model="activeTab">
      <b-tab-item :label="$t('settings.appearance.admin.name')" label-position="on-border">
        <div class="block">
          <p>
            {{ $t('settings.appearance.admin.help') }}
          </p>
        </div>
        <br /><br />

        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border"
        :message="$t('settings.appearance.cssHelp')">
          <appearance-editor v-model="data['appearance.admin.custom_css']" name="body"
          language="css" />
        </b-field>
      </b-tab-item><!-- admin -->

      <b-tab-item :label="$t('settings.appearance.public.name')" label-position="on-border">
        <div class="block">
          <p>
            {{ $t('settings.appearance.public.help') }}
          </p>
        </div>
        <br /><br />

        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border"
        :message="$t('settings.appearance.cssHelp')">
          <appearance-editor v-model="data['appearance.public.custom_css']" name="body"
          language="css" />
        </b-field>

        <b-field :label="$t('settings.appearance.customJS')" label-position="on-border"
        :message="$t('settings.appearance.jsHelp')">
          <appearance-editor v-model="data['appearance.public.custom_js']" name="body"
          language="javascript" />
        </b-field>
      </b-tab-item><!-- public -->
    </b-tabs>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import AppearanceEditor from '../../components/AppearanceEditor.vue';

export default Vue.extend({
  components: {
    'appearance-editor': AppearanceEditor,
  },

  props: {
    form: {
      type: Object,
    },
  },

  data() {
    return {
      data: this.form,
      activeTab: 0,
    };
  },

  watch: {
    activeTab: function activeTab() {
      localStorage.setItem('admin.settings.appearance.active_tab', this.activeTab);
    },
  },

  mounted() {
    // Reload active tab.
    if (localStorage.getItem('admin.settings.appearance.active_tab')) {
      this.activeTab = JSON.parse(localStorage.getItem('admin.settings.appearance.active_tab'));
    }
  },

  computed: {
    ...mapState(['settings']),
  },
});

</script>
