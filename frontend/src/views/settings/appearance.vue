<template>
  <div class="items">
    <b-tabs type="is-boxed" :animated="false" v-model="activeTab">
      <!-- eslint-disable-next-line max-len -->
      <b-tab-item :label="$t('settings.appearance.admin.name')" label-position="on-border">
        <div class="block">
          <p>
            {{ $t('settings.appearance.admin.help') }}
          </p>
        </div>
        <br /><br />

        <!-- eslint-disable-next-line max-len -->
        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border" :message="$t('settings.appearance.cssHelp')">
          <!-- eslint-disable-next-line max-len -->
          <b-input v-model="data['appearance.admin.custom_css']" type="textarea" name="body" />
        </b-field>
      </b-tab-item><!-- admin -->

      <!-- eslint-disable-next-line max-len -->
      <b-tab-item :label="$t('settings.appearance.public.name')" label-position="on-border">
        <div class="block">
          <p>
            {{ $t('settings.appearance.public.help') }}
          </p>
        </div>
        <br /><br />

        <!-- eslint-disable-next-line max-len -->
        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border" :message="$t('settings.appearance.cssHelp')">
          <!-- eslint-disable-next-line max-len -->
          <b-input v-model="data['appearance.public.custom_css']" type="textarea" name="body" />
        </b-field>

        <!-- eslint-disable-next-line max-len -->
        <b-field :label="$t('settings.appearance.customJS')" label-position="on-border" :message="$t('settings.appearance.jsHelp')">
          <!-- eslint-disable-next-line max-len -->
          <b-input v-model="data['appearance.public.custom_js']" type="textarea" name="body" />
        </b-field>
      </b-tab-item><!-- public -->
    </b-tabs>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
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
