<template>
  <div class="items">
    <div class="columns">
      <div class="column is-4">
        <b-field :label="$t('settings.security.enableOIDC')" :message="$t('settings.security.OIDCHelp')">
          <b-switch v-model="data['security.oidc']['enabled']" name="security.oidc" />
        </b-field>
      </div>
      <div class="column is-8">
        <b-field :label="$t('settings.security.OIDCURL')" label-position="on-border">
          <div>
            <b-input v-model="data['security.oidc']['provider_url']" name="oidc.provider_url"
              placeholder="https://login.yoursite.com" :disabled="!data['security.oidc']['enabled']" :maxlength="300"
              required type="url" pattern="https?://.*" />

            <div class="spaced-links is-size-7 mt-2" :class="{ 'disabled': !data['security.oidc']['enabled'] }">
              <a href="#" @click.prevent="() => setProvider(n, 'google')">Google</a>
              <a href="#" @click.prevent="() => setProvider(n, 'github')">GitHub</a>
              <a href="#" @click.prevent="() => setProvider(n, 'microsoft')">Microsoft</a>
              <a href="#" @click.prevent="() => setProvider(n, 'apple')">Apple</a>
            </div>
          </div>
        </b-field>

        <b-field :label="$t('settings.security.OIDCClientID')" label-position="on-border">
          <b-input v-model="data['security.oidc']['client_id']" name="oidc.client_id" ref="client_id"
            :disabled="!data['security.oidc']['enabled']" :maxlength="200" required />
        </b-field>

        <b-field :label="$t('settings.security.OIDCClientSecret')" label-position="on-border">
          <b-input v-model="data['security.oidc']['client_secret']" name="oidc.client_secret" type="password"
            :disabled="!data['security.oidc']['enabled']" :maxlength="200" required />
        </b-field>

        <b-field :label="$t('settings.security.OIDCRedirectURL')">
          <code><copy-text :text="`${serverConfig.root_url}/auth/oidc`" /></code>
        </b-field>
      </div>
    </div>

    <hr />
    <div class="columns">
      <div class="column is-4">
        <b-field :label="$t('settings.security.enableCaptcha')" :message="$t('settings.security.enableCaptchaHelp')">
          <b-switch v-model="data['security.enable_captcha']" name="security.captcha" />
        </b-field>
      </div>
      <div class="column is-8">
        <b-field :label="$t('settings.security.captchaKey')" label-position="on-border"
          :message="$t('settings.security.captchaKeyHelp')">
          <b-input v-model="data['security.captcha_key']" name="captcha_key"
            :disabled="!data['security.enable_captcha']" :maxlength="200" required />
        </b-field>
        <b-field :label="$t('settings.security.captchaSecret')" label-position="on-border">
          <b-input v-model="data['security.captcha_secret']" name="captcha_secret" type="password"
            :disabled="!data['security.enable_captcha']" :maxlength="200" required />
        </b-field>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CopyText from '../../components/CopyText.vue';

const OIDC_PROVIDERS = {
  google: 'https://accounts.google.com',
  github: 'https://token.actions.githubusercontent.com',
  microsoft: 'https://login.microsoftonline.com/{TENANT_HERE}/v2.0',
  apple: 'https://appleid.apple.com',
};

export default Vue.extend({
  components: {
    CopyText,
  },

  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  computed: {
    ...mapState(['serverConfig']),

    version() {
      return import.meta.env.VUE_APP_VERSION;
    },

    isMobile() {
      return this.windowWidth <= 768;
    },
  },

  methods: {
    setProvider(n, provider) {
      this.$set(this.data['security.oidc'], 'provider_url', OIDC_PROVIDERS[provider]);

      this.$nextTick(() => {
        this.$refs.client_id.focus();
      });
    },
  },

  data() {
    return {
      data: this.form,
    };
  },
});
</script>
