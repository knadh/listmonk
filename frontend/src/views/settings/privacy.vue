<template>
  <div class="items">
    <b-field :label="$t('settings.privacy.individualSubTracking')"
      :message="isExternallyManaged('privacy.individual_tracking') ? 'This setting is configured externally' : $t('settings.privacy.individualSubTrackingHelp')">
      <b-switch v-model="data['privacy.individual_tracking']" name="privacy.individual_tracking" :disabled="isExternallyManaged('privacy.individual_tracking')" />
    </b-field>

    <b-field :label="$t('settings.privacy.listUnsubHeader')"
      :message="isExternallyManaged('privacy.unsubscribe_header') ? 'This setting is configured externally' : $t('settings.privacy.listUnsubHeaderHelp')">
      <b-switch v-model="data['privacy.unsubscribe_header']" name="privacy.unsubscribe_header" :disabled="isExternallyManaged('privacy.unsubscribe_header')" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowBlocklist')"
      :message="isExternallyManaged('privacy.allow_blocklist') ? 'This setting is configured externally' : $t('settings.privacy.allowBlocklistHelp')">
      <b-switch v-model="data['privacy.allow_blocklist']" name="privacy.allow_blocklist" :disabled="isExternallyManaged('privacy.allow_blocklist')" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowPrefs')"
      :message="isExternallyManaged('privacy.allow_preferences') ? 'This setting is configured externally' : $t('settings.privacy.allowPrefsHelp')">
      <b-switch v-model="data['privacy.allow_preferences']" name="privacy.allow_preferences" :disabled="isExternallyManaged('privacy.allow_preferences')" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowExport')"
      :message="isExternallyManaged('privacy.allow_export') ? 'This setting is configured externally' : $t('settings.privacy.allowExportHelp')">
      <b-switch v-model="data['privacy.allow_export']" name="privacy.allow_export" :disabled="isExternallyManaged('privacy.allow_export')" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowWipe')"
      :message="isExternallyManaged('privacy.allow_wipe') ? 'This setting is configured externally' : $t('settings.privacy.allowWipeHelp')">
      <b-switch v-model="data['privacy.allow_wipe']" name="privacy.allow_wipe" :disabled="isExternallyManaged('privacy.allow_wipe')" />
    </b-field>

    <b-field :label="$t('settings.privacy.recordOptinIP')"
      :message="isExternallyManaged('privacy.record_optin_ip') ? 'This setting is configured externally' : $t('settings.privacy.recordOptinIPHelp')">
      <b-switch v-model="data['privacy.record_optin_ip']" name="privacy.record_optin_ip" :disabled="isExternallyManaged('privacy.record_optin_ip')" />
    </b-field>

    <hr />

    <b-tabs v-model="tab" type="is-boxed" :animated="false">
      <b-tab-item :label="`${$t('settings.privacy.domainBlocklist')} (${numBlocked})`">
        <b-field :message="$t('settings.privacy.domainBlocklistHelp')">
          <b-input type="textarea" v-model="data['privacy.domain_blocklist']" name="privacy.domain_blocklist" />
        </b-field>
      </b-tab-item>
      <b-tab-item :label="`${$t('settings.privacy.domainAllowlist')} (${numAllowed})`">
        <b-field :message="$t('settings.privacy.domainAllowlistHelp')">
          <b-input type="textarea" v-model="data['privacy.domain_allowlist']" name="privacy.domain_allowlist" />
        </b-field>
      </b-tab-item>
    </b-tabs>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
    externalSettings: {
      type: Array, default: () => [],
    },
  },

  data() {
    return {
      data: this.form,
      tab: 0,
    };
  },

  methods: {
    isExternallyManaged(settingKey) {
      return this.externalSettings.includes(settingKey);
    },

    countItems(str) {
      return str.split('\n').filter((line) => line.trim()).length;
    },
  },

  mounted() {
    this.tab = this.$utils.getPref('settings.privacyDomainTab') || 0;
  },

  computed: {
    numBlocked() {
      return this.countItems(this.form['privacy.domain_blocklist']);
    },
    numAllowed() {
      return this.countItems(this.form['privacy.domain_allowlist']);
    },
  },

  watch: {
    tab(t) {
      this.$utils.setPref('settings.privacyDomainTab', t);
    },
  },
});
</script>
