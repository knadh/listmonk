<template>
  <div class="items">
    <oat-field :label="$t('settings.general.siteName')">
      <input aria-label="field" v-model="data['app.site_name']" name="app.site_name" :label="$t('settings.general.siteName')"
        :maxlength="300" required>
    </oat-field>

    <oat-field :label="$t('settings.general.rootURL')"
      :message="$t('settings.general.rootURLHelp')">
      <input aria-label="field" v-model="data['app.root_url']" name="app.root_url" placeholder="https://listmonk.yoursite.com"
        :maxlength="300" required type="url" pattern="https?://.*">
    </oat-field>

    <oat-field :label="$t('settings.general.logoURL')"
      :message="$t('settings.general.logoURLHelp')">
      <input aria-label="field" v-model="data['app.logo_url']" name="app.logo_url" placeholder="https://listmonk.yoursite.com/logo.png"
        :maxlength="300" type="url" pattern="https?://.*">
    </oat-field>
    <oat-field :label="$t('settings.general.faviconURL')"
      :message="$t('settings.general.faviconURLHelp')">
      <input aria-label="field" v-model="data['app.favicon_url']" name="app.favicon_url"
        placeholder="https://listmonk.yoursite.com/favicon.png" :maxlength="300" type="url" pattern="https?://.*">
    </oat-field>

    <hr />
    <oat-field :label="$t('settings.general.fromEmail')"
      :message="$t('settings.general.fromEmailHelp')">
      <input aria-label="field" v-model="data['app.from_email']" name="app.from_email"
        placeholder="Listmonk <noreply@listmonk.yoursite.com>" pattern="((.+?)\s)?<(.+?)@(.+?)>" :maxlength="300" />
    </oat-field>
    <oat-field :label="$t('settings.general.adminNotifEmails')"
      :message="$t('settings.general.adminNotifEmailsHelp')">
      <oat-tag-input v-model="data['app.notify_emails']" name="app.notify_emails"
        :before-adding="(v) => v.match(/(.+?)@(.+?)/)" placeholder="you@yoursite.com" />
    </oat-field>

    <hr />

    <div>
      <h2 class="text-4 mb-5">
        {{ $tc('globals.terms.subscriptions', 2) }}
      </h2>
      <oat-field :message="$t('settings.general.enablePublicSubPageHelp')">
        <oat-switch v-model="data['app.enable_public_subscription_page']" name="app.enable_public_subscription_page">
          {{ $t('settings.general.enablePublicSubPage') }}
        </oat-switch>
      </oat-field>
      <oat-field :message="$t('settings.general.sendOptinConfirmHelp')">
        <oat-switch v-model="data['app.send_optin_confirmation']" name="app.send_optin_confirmation">
          {{ $t('settings.general.sendOptinConfirm') }}
        </oat-switch>
      </oat-field>
      <oat-field :message="$t('settings.general.showOptinPageHelp')">
        <oat-switch v-model="data['app.show_optin_page']" name="app.show_optin_page">
          {{ $t('settings.general.showOptinPage') }}
        </oat-switch>
      </oat-field>
    </div>
    <hr />

    <div>
      <h2 class="text-4 mb-5">
        {{ $t('campaigns.archive') }}
      </h2>
      <oat-field :message="$t('settings.general.enablePublicArchiveHelp')">
        <oat-switch v-model="data['app.enable_public_archive']" name="app.enable_public_archive">
          {{ $t('settings.general.enablePublicArchive') }}
        </oat-switch>
      </oat-field>
      <oat-field :message="$t('settings.general.enablePublicArchiveRSSContentHelp')">
        <oat-switch v-model="data['app.enable_public_archive_rss_content']" name="app.enable_public_archive_rss_content">
          {{ $t('settings.general.enablePublicArchiveRSSContent') }}
        </oat-switch>
      </oat-field>
    </div>

    <hr />
    <oat-field :message="$t('settings.general.checkUpdatesHelp')">
      <oat-switch v-model="data['app.check_updates']" name="app.check_updates">
        {{ $t('settings.general.checkUpdates') }}
      </oat-switch>
    </oat-field>

    <hr />
    <oat-field :label="$t('settings.general.language')">
      <select aria-label="field" v-model="data['app.lang']" name="app.lang">
        <option v-for="l in serverConfig.langs" :key="l.code" :value="l.code">
          {{ l.name }}
        </option>
      </select>
      <p class="mt-2">
        <a href="https://listmonk.app/docs/i18n/#additional-language-packs" target="_blank" rel="noopener noreferer">{{
          $t('globals.buttons.more') }} &rarr;</a>
      </p>
    </oat-field>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
    };
  },

  computed: {
    ...mapState(['serverConfig', 'loading']),
  },

});
</script>
