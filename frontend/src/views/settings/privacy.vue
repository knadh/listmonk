<template>
  <div class="items">
    <div class="columns">
      <div class="column is-6">
        <b-field :label="$t('settings.privacy.disableTracking')" :message="$t('settings.privacy.disableTrackingHelp')">
          <b-switch v-model="data['privacy.disable_tracking']" name="privacy.disable_tracking" />
        </b-field>
      </div>
      <div class="column is-6" :class="{ 'is-disabled': data['privacy.disable_tracking'] }">
        <b-field :label="$t('settings.privacy.individualSubTracking')"
          :message="$t('settings.privacy.individualSubTrackingHelp')">
          <b-switch v-model="data['privacy.individual_tracking']" :disabled="data['privacy.disable_tracking']"
            name="privacy.individual_tracking" />
        </b-field>
      </div>
    </div>

    <b-field :label="$t('settings.privacy.listUnsubHeader')" :message="$t('settings.privacy.listUnsubHeaderHelp')">
      <b-switch v-model="data['privacy.unsubscribe_header']" name="privacy.unsubscribe_header" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowBlocklist')" :message="$t('settings.privacy.allowBlocklistHelp')">
      <b-switch v-model="data['privacy.allow_blocklist']" name="privacy.allow_blocklist" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowPrefs')" :message="$t('settings.privacy.allowPrefsHelp')">
      <b-switch v-model="data['privacy.allow_preferences']" name="privacy.allow_blocklist" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowExport')" :message="$t('settings.privacy.allowExportHelp')">
      <b-switch v-model="data['privacy.allow_export']" name="privacy.allow_export" />
    </b-field>

    <b-field :label="$t('settings.privacy.allowWipe')" :message="$t('settings.privacy.allowWipeHelp')">
      <b-switch v-model="data['privacy.allow_wipe']" name="privacy.allow_wipe" />
    </b-field>

    <b-field :label="$t('settings.privacy.recordOptinIP')" :message="$t('settings.privacy.recordOptinIPHelp')">
      <b-switch v-model="data['privacy.record_optin_ip']" name="privacy.record_optin_ip" />
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
  },

  data() {
    return {
      data: this.form,
      tab: 0,
    };
  },

  methods: {
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
