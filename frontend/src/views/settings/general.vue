<template>
  <div class="items">
    <b-field :label="$t('settings.general.siteName')" label-position="on-border">
      <b-input v-model="data['app.site_name']" name="app.site_name"
          :label="$t('settings.general.siteName')" :maxlength="300" required />
    </b-field>

    <b-field :label="$t('settings.general.rootURL')" label-position="on-border"
      :message="$t('settings.general.rootURLHelp')">
      <b-input v-model="data['app.root_url']" name="app.root_url"
          placeholder='https://listmonk.yoursite.com' :maxlength="300" required />
    </b-field>

    <div class="columns">
      <div class="column is-6">
        <b-field :label="$t('settings.general.logoURL')" label-position="on-border"
          :message="$t('settings.general.logoURLHelp')">
          <b-input v-model="data['app.logo_url']" name="app.logo_url"
              placeholder='https://listmonk.yoursite.com/logo.png' :maxlength="300" />
        </b-field>
      </div>
      <div class="column is-6">
        <b-field :label="$t('settings.general.faviconURL')" label-position="on-border"
          :message="$t('settings.general.faviconURLHelp')">
          <b-input v-model="data['app.favicon_url']" name="app.favicon_url"
              placeholder='https://listmonk.yoursite.com/favicon.png' :maxlength="300" />
        </b-field>
      </div>
    </div>

    <hr />
    <b-field :label="$t('settings.general.fromEmail')" label-position="on-border"
      :message="$t('settings.general.fromEmailHelp')">
      <b-input v-model="data['app.from_email']" name="app.from_email"
          placeholder='Listmonk <noreply@listmonk.yoursite.com>'
          pattern="(.+?)\s<(.+?)@(.+?)>" :maxlength="300" />
    </b-field>
    <b-field :label="$t('settings.general.adminNotifEmails')" label-position="on-border"
      :message="$t('settings.general.adminNotifEmailsHelp')">
      <b-taginput v-model="data['app.notify_emails']" name="app.notify_emails"
        :before-adding="(v) => v.match(/(.+?)@(.+?)/)"
        placeholder='you@yoursite.com' />
    </b-field>

    <hr />
    <div class="columns">
      <div class="column is-4">
        <b-field :label="$t('settings.general.enablePublicSubPage')"
          :message="$t('settings.general.enablePublicSubPageHelp')">
          <b-switch v-model="data['app.enable_public_subscription_page']"
              name="app.enable_public_subscription_page" />
        </b-field>
      </div>
      <div class="column is-4">
        <b-field :label="$t('settings.general.enablePublicArchive')"
          :message="$t('settings.general.enablePublicArchiveHelp')">
          <b-switch v-model="data['app.enable_public_archive']"
              name="app.enable_public_archive" />
        </b-field>
      </div>
      <div class="column is-4">
        <b-field :label="$t('settings.general.sendOptinConfirm')"
          :message="$t('settings.general.sendOptinConfirmHelp')">
          <b-switch v-model="data['app.send_optin_confirmation']"
              name="app.send_optin_confirmation" />
        </b-field>
      </div>
    </div>

    <hr />
    <b-field :label="$t('settings.general.checkUpdates')"
      :message="$t('settings.general.checkUpdatesHelp')">
      <b-switch v-model="data['app.check_updates']"
          name="app.check_updates" />
    </b-field>

    <hr />
    <b-field :label="$t('settings.general.language')" label-position="on-border" :addons="false">
      <b-select v-model="data['app.lang']" name="app.lang">
          <option v-for="l in serverConfig.langs" :key="l.code" :value="l.code">
            {{ l.name }}
          </option>
      </b-select>
      <p class="mt-2">
        <a href="https://listmonk.app/docs/i18n/#additional-language-packs" target="_blank">{{ $t('globals.buttons.more') }} &rarr;</a>
      </p>
    </b-field>
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
    };
  },

  computed: {
    ...mapState(['serverConfig', 'loading']),
  },

});
</script>
