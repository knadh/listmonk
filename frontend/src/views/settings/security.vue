<template>
  <div class="items">
    <div class="row">
      <div class="col-3">
        <b-field :message="$t('settings.security.OIDCHelp')">
          <b-switch v-model="data['security.oidc']['enabled']" name="security.oidc">
            {{ $t('settings.security.enableOIDC') }}
          </b-switch>
        </b-field>
      </div>
      <div class="col-9">
        <b-field :label="$t('settings.security.OIDCURL')">
          <div>
            <input aria-label="field" v-model="data['security.oidc']['provider_url']" name="oidc.provider_url"
              placeholder="https://login.yoursite.com" :disabled="!data['security.oidc']['enabled']" :maxlength="300"
              required type="url" pattern="https?://.*">

            <div class="spaced-links text-7 mt-2" :class="{ disabled: !data['security.oidc']['enabled'] }">
              <a href="#" @click.prevent="() => setProvider('google')">Google</a>
              <a href="#" @click.prevent="() => setProvider('microsoft')">Microsoft</a>
              <a href="#" @click.prevent="() => setProvider('apple')">Apple</a>
            </div>
          </div>
        </b-field>

        <b-field :label="$t('settings.security.OIDCName')">
          <input aria-label="field" v-model="data['security.oidc']['provider_name']" name="oidc.provider_name"
            ref="provider_name" :disabled="!data['security.oidc']['enabled']" :maxlength="200">
        </b-field>

        <b-field :label="$t('settings.security.OIDCClientID')">
          <input aria-label="field" v-model="data['security.oidc']['client_id']" name="oidc.client_id" ref="client_id"
            :disabled="!data['security.oidc']['enabled']" :maxlength="200" required>
        </b-field>

        <b-field :label="$t('settings.security.OIDCClientSecret')">
          <input aria-label="field" v-model="data['security.oidc']['client_secret']" name="oidc.client_secret"
            type="password" :disabled="!data['security.oidc']['enabled']" :maxlength="200" required>
        </b-field>

        <hr />

        <b-field :message="$t('settings.security.OIDCAutoCreateUsersHelp')">
          <b-switch v-model="data['security.oidc']['auto_create_users']" :disabled="!data['security.oidc']['enabled']"
            name="oidc.auto_create_users">
            {{ $t('settings.security.OIDCAutoCreateUsers') }}
          </b-switch>
        </b-field>

        <b-field :label="$t('settings.security.OIDCDefaultUserRole')"
          :message="$t('settings.security.OIDCDefaultRoleHelp')">
          <select aria-label="field" v-model="data['security.oidc']['default_user_role_id']"
            :disabled="!data['security.oidc']['enabled'] || !data['security.oidc']['auto_create_users']"
            name="oidc.default_user_role_id">
            <option v-for="role in userRoles" :key="role.id" :value="role.id">
              {{ role.name }}
            </option>
          </select>
        </b-field>

        <b-field :label="$t('settings.security.OIDCDefaultListRole')"
          :message="$t('settings.security.OIDCDefaultRoleHelp')">
          <select aria-label="field" v-model="data['security.oidc']['default_list_role_id']"
            :disabled="!data['security.oidc']['enabled'] || !data['security.oidc']['auto_create_users']"
            name="oidc.default_list_role_id">
            <option :value="null">&mdash; {{ $t("globals.terms.none") }} &mdash;</option>
            <option v-for="role in listRoles" :key="role.id" :value="role.id">
              {{ role.name }}
            </option>
          </select>
        </b-field>

        <hr />

        <b-field :label="$t('settings.security.OIDCRedirectURL')">
          <code><copy-text :text="`${serverConfig.root_url}/auth/oidc`" /></code>
        </b-field>
        <p v-if="data['security.oidc']['enabled'] && !isURLOk" class="text-danger">
          <b-icon icon="warning-empty" />
          {{ $t('settings.security.OIDCRedirectWarning') }}
        </p>
      </div>
    </div>

    <hr />
    <div class="row">
      <div class="col-3">
        <b-field :message="$t('settings.security.enableCaptchaHelp')">
          <b-switch v-model="captchaEnabled" name="security.captcha">
            {{ $t('settings.security.enableCaptcha') }}
          </b-switch>
        </b-field>
      </div>
      <div class="col-9" v-if="captchaEnabled">
        <fieldset>
          <legend>{{ $t('settings.security.enableCaptcha') }}</legend>
          <label>
            <input aria-label="field" v-model="selectedProvider" type="radio" value="altcha" name="captcha_provider">
            ALTCHA
          </label>
          <label>
            <input aria-label="field" v-model="selectedProvider" type="radio" value="hcaptcha" name="captcha_provider">
            hCaptcha (deprecated)
          </label>
        </fieldset>

        <!-- captcha settings -->
        <div v-if="selectedProvider === 'altcha'">
          <b-field :label="$t('settings.security.altchaComplexity')"
            :message="$t('settings.security.altchaComplexityHelp')">
            <input aria-label="field" v-model.number="data['security.captcha']['altcha']['complexity']"
              name="altcha_complexity" type="number" min="1000" max="1000000" required>
          </b-field>
        </div>
        <div v-if="selectedProvider === 'hcaptcha'">
          <b-field :label="$t('settings.security.captchaKey')" :message="$t('settings.security.captchaKeyHelp')">
            <input aria-label="field" v-model="data['security.captcha']['hcaptcha']['key']" name="hcaptcha_key"
              :maxlength="200" required>
          </b-field>
          <b-field :label="$t('settings.security.captchaSecret')">
            <input aria-label="field" v-model="data['security.captcha']['hcaptcha']['secret']" name="hcaptcha_secret"
              type="password" :maxlength="200" required>
          </b-field>
        </div>
      </div>
    </div><!-- captcha -->

    <hr />

    <!-- CORS -->
    <div class="row">
      <div class="col-12">
        <h3><strong>{{ $t('settings.security.trustedURLs') }} / CORS</strong></h3><br />
        <b-field :message="$t('settings.security.trustedURLsHelp')">
          <textarea aria-label="field" v-model="trustedURLs" name="trusted_urls" rows="5"
            placeholder="https://example.com" />
        </b-field>
      </div>
    </div><!-- cors -->
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
    ...mapState(['serverConfig', 'userRoles', 'listRoles']),

    trustedURLs: {
      get() {
        // Convert array to newline-separated string.
        const domains = this.data['security.trusted_urls'];
        return domains && Array.isArray(domains) ? domains.join('\n') : '';
      },
      set(value) {
        this.$set(this.data, 'security.trusted_urls', value.split('\n'));
      },
    },

    captchaEnabled: {
      get() {
        return this.data['security.captcha'].altcha.enabled || this.data['security.captcha'].hcaptcha.enabled;
      },
      set(value) {
        this.data['security.captcha'].altcha.enabled = !!value;
        this.data['security.captcha'].hcaptcha.enabled = false;
      },
    },

    selectedProvider: {
      get() {
        if (this.data['security.captcha'].hcaptcha.enabled) {
          return 'hcaptcha';
        }

        return 'altcha';
      },
      set(value) {
        this.data['security.captcha'].hcaptcha.enabled = value === 'hcaptcha';
        this.data['security.captcha'].altcha.enabled = value === 'altcha';
      },
    },

    version() {
      return import.meta.env.VUE_APP_VERSION;
    },

    isMobile() {
      return this.windowWidth <= 768;
    },

    isURLOk() {
      try {
        const u = new URL(this.serverConfig.root_url);
        return u.hostname !== 'localhost' && u.hostname !== '127.0.0.1';
      } catch (e) {
        return false;
      }
    },
  },

  mounted() {
    if (this.$can('roles:get')) {
      this.$api.getUserRoles();
      this.$api.getListRoles();
    }
  },

  methods: {
    setProvider(provider) {
      this.$set(this.data['security.oidc'], 'provider_url', OIDC_PROVIDERS[provider]);
      this.$set(this.data['security.oidc'], 'provider_name', provider.charAt(0).toUpperCase() + provider.slice(1));

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
