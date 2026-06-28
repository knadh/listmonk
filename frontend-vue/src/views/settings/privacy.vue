<template>
  <div class="items">
    <div class="row">
      <div class="col-6">
        <b-field :message="$t('settings.privacy.disableTrackingHelp')">
          <b-switch v-model="data['privacy.disable_tracking']" name="privacy.disable_tracking">
            {{ $t('settings.privacy.disableTracking') }}
          </b-switch>
        </b-field>
      </div>
      <div class="col-6" :class="{ disabled: data['privacy.disable_tracking'] }">
        <b-field :message="$t('settings.privacy.individualSubTrackingHelp')">
          <b-switch v-model="data['privacy.individual_tracking']" :disabled="data['privacy.disable_tracking']"
            name="privacy.individual_tracking">
            {{ $t('settings.privacy.individualSubTracking') }}
          </b-switch>
        </b-field>
      </div>
    </div>

    <b-field :message="$t('settings.privacy.listUnsubHeaderHelp')">
      <b-switch v-model="data['privacy.unsubscribe_header']" name="privacy.unsubscribe_header">
        {{ $t('settings.privacy.listUnsubHeader') }}
      </b-switch>
    </b-field>

    <b-field :message="$t('settings.privacy.allowBlocklistHelp')">
      <b-switch v-model="data['privacy.allow_blocklist']" name="privacy.allow_blocklist">
        {{ $t('settings.privacy.allowBlocklist') }}
      </b-switch>
    </b-field>

    <b-field :message="$t('settings.privacy.allowPrefsHelp')">
      <b-switch v-model="data['privacy.allow_preferences']" name="privacy.allow_blocklist">
        {{ $t('settings.privacy.allowPrefs') }}
      </b-switch>
    </b-field>

    <b-field :message="$t('settings.privacy.allowExportHelp')">
      <b-switch v-model="data['privacy.allow_export']" name="privacy.allow_export">
        {{ $t('settings.privacy.allowExport') }}
      </b-switch>
    </b-field>

    <b-field :message="$t('settings.privacy.allowWipeHelp')">
      <b-switch v-model="data['privacy.allow_wipe']" name="privacy.allow_wipe">
        {{ $t('settings.privacy.allowWipe') }}
      </b-switch>
    </b-field>

    <b-field :message="$t('settings.privacy.recordOptinIPHelp')">
      <b-switch v-model="data['privacy.record_optin_ip']" name="privacy.record_optin_ip">
        {{ $t('settings.privacy.recordOptinIP') }}
      </b-switch>
    </b-field>

    <hr />

    <ot-tabs ref="privacyDomainTabs" class="settings-subtabs" @ot-tab-change="tab = $event.detail.index">
      <div role="tablist">
        <button type="button" role="tab" :aria-selected="tab === 0 ? 'true' : 'false'">
          {{ `${$t('settings.privacy.domainBlocklist')} (${numBlocked})` }}
        </button>
        <button type="button" role="tab" :aria-selected="tab === 1 ? 'true' : 'false'">
          {{ `${$t('settings.privacy.domainAllowlist')} (${numAllowed})` }}
        </button>
      </div>

      <div role="tabpanel">
        <b-field :message="$t('settings.privacy.domainBlocklistHelp')">
          <textarea aria-label="field" v-model="data['privacy.domain_blocklist']" name="privacy.domain_blocklist" />
        </b-field>
      </div>
      <div role="tabpanel">
        <b-field :message="$t('settings.privacy.domainAllowlistHelp')">
          <textarea aria-label="field" v-model="data['privacy.domain_allowlist']" name="privacy.domain_allowlist" />
        </b-field>
      </div>
    </ot-tabs>
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
    this.$nextTick(() => {
      if (this.$refs.privacyDomainTabs) {
        this.$refs.privacyDomainTabs.activeIndex = Number(this.tab);
      }
    });
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
