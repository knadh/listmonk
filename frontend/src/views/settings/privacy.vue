<template>
  <div class="items">
    <div class="row">
      <div class="col-6">
        <oat-field :message="$t('settings.privacy.disableTrackingHelp')">
          <oat-switch v-model="data['privacy.disable_tracking']" name="privacy.disable_tracking">
            {{ $t('settings.privacy.disableTracking') }}
          </oat-switch>
        </oat-field>
      </div>
      <div class="col-6" :class="{ disabled: data['privacy.disable_tracking'] }">
        <oat-field :message="$t('settings.privacy.individualSubTrackingHelp')">
          <oat-switch v-model="data['privacy.individual_tracking']" :disabled="data['privacy.disable_tracking']"
            name="privacy.individual_tracking">
            {{ $t('settings.privacy.individualSubTracking') }}
          </oat-switch>
        </oat-field>
      </div>
    </div>

    <oat-field :message="$t('settings.privacy.listUnsubHeaderHelp')">
      <oat-switch v-model="data['privacy.unsubscribe_header']" name="privacy.unsubscribe_header">
        {{ $t('settings.privacy.listUnsubHeader') }}
      </oat-switch>
    </oat-field>

    <oat-field :message="$t('settings.privacy.allowBlocklistHelp')">
      <oat-switch v-model="data['privacy.allow_blocklist']" name="privacy.allow_blocklist">
        {{ $t('settings.privacy.allowBlocklist') }}
      </oat-switch>
    </oat-field>

    <oat-field :message="$t('settings.privacy.allowPrefsHelp')">
      <oat-switch v-model="data['privacy.allow_preferences']" name="privacy.allow_blocklist">
        {{ $t('settings.privacy.allowPrefs') }}
      </oat-switch>
    </oat-field>

    <oat-field :message="$t('settings.privacy.allowExportHelp')">
      <oat-switch v-model="data['privacy.allow_export']" name="privacy.allow_export">
        {{ $t('settings.privacy.allowExport') }}
      </oat-switch>
    </oat-field>

    <oat-field :message="$t('settings.privacy.allowWipeHelp')">
      <oat-switch v-model="data['privacy.allow_wipe']" name="privacy.allow_wipe">
        {{ $t('settings.privacy.allowWipe') }}
      </oat-switch>
    </oat-field>

    <oat-field :message="$t('settings.privacy.recordOptinIPHelp')">
      <oat-switch v-model="data['privacy.record_optin_ip']" name="privacy.record_optin_ip">
        {{ $t('settings.privacy.recordOptinIP') }}
      </oat-switch>
    </oat-field>

    <hr />

    <div class="settings-subtabs">
      <div role="tablist">
        <button type="button" role="tab" :aria-selected="tab === 0 ? 'true' : 'false'"
          :class="{ outline: tab !== 0 }" @click="tab = 0">
          {{ `${$t('settings.privacy.domainBlocklist')} (${numBlocked})` }}
        </button>
        <button type="button" role="tab" :aria-selected="tab === 1 ? 'true' : 'false'"
          :class="{ outline: tab !== 1 }" @click="tab = 1">
          {{ `${$t('settings.privacy.domainAllowlist')} (${numAllowed})` }}
        </button>
      </div>

      <div v-show="tab === 0" role="tabpanel">
        <oat-field :message="$t('settings.privacy.domainBlocklistHelp')">
          <textarea aria-label="field" v-model="data['privacy.domain_blocklist']" name="privacy.domain_blocklist" />
        </oat-field>
      </div>
      <div v-show="tab === 1" role="tabpanel">
        <oat-field :message="$t('settings.privacy.domainAllowlistHelp')">
          <textarea aria-label="field" v-model="data['privacy.domain_allowlist']" name="privacy.domain_allowlist" />
        </oat-field>
      </div>
    </div>
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
