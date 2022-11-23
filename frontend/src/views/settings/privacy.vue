<template>
  <div class="items">
    <b-field :label="$t('settings.privacy.individualSubTracking')"
      :message="$t('settings.privacy.individualSubTrackingHelp')">
      <b-switch v-model="data['privacy.individual_tracking']"
          name="privacy.individual_tracking" />
    </b-field>

    <b-field :label="$t('settings.privacy.listUnsubHeader')"
      :message="$t('settings.privacy.listUnsubHeaderHelp')">
      <b-switch v-model="data['privacy.unsubscribe_header']"
          name="privacy.unsubscribe_header" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowBlocklist')"
      :message="$t('settings.privacy.allowBlocklistHelp')">
      <b-switch v-model="data['privacy.allow_blocklist']"
          name="privacy.allow_blocklist" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowPrefs')"
      :message="$t('settings.privacy.allowPrefsHelp')">
      <b-switch v-model="data['privacy.allow_preferences']"
          name="privacy.allow_blocklist" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowExport')"
      :message="$t('settings.privacy.allowExportHelp')">
      <b-switch v-model="data['privacy.allow_export']"
          name="privacy.allow_export" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowWipe')"
      :message="$t('settings.privacy.allowWipeHelp')">
      <b-switch v-model="data['privacy.allow_wipe']"
          name="privacy.allow_wipe" />
    </b-field>

    <b-field :label="$t('settings.privacy.domainBlocklist')"
      :message="$t('settings.privacy.domainBlocklistHelp')">
      <b-input type="textarea"
          v-model="data['privacy.domain_blocklist']"
          name="privacy.domain_blocklist" />
    </b-field>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
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
});
</script>
