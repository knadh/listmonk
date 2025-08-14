<template>
  <div class="items">
    <div class="columns">
      <div class="column is-3">
        <b-field :label="$t('settings.security.enableOIDC')" :message="$t('settings.security.OIDCHelp')">
          <b-switch v-model="data['security.oidc']['enabled']" name="security.oidc" />
        </b-field>
      </div>
      <div class="column is-9">
        <div class="columns">
          <div class="column is-7">
            <b-field :label="$t('settings.security.OIDCURL')" label-position="on-border">
              <div>
                <b-input v-model="data['security.oidc']['provider_url']" name="oidc.provider_url"
                  placeholder="https://login.yoursite.com" :disabled="!data['security.oidc']['enabled']"
                  :maxlength="300" required type="url" pattern="https?://.*" />

                <div class="spaced-links is-size-7 mt-2" :class="{ 'disabled': !data['security.oidc']['enabled'] }">
                  <a href="#" @click.prevent="() => setProvider('google')">Google</a>
                  <a href="#" @click.prevent="() => setProvider('microsoft')">Microsoft</a>
                  <a href="#" @click.prevent="() => setProvider('apple')">Apple</a>
                </div>
              </div>
            </b-field>
          </div>
          <div class="column is-5">
            <b-field :label="$t('settings.security.OIDCName')" label-position="on-border">
              <b-input v-model="data['security.oidc']['provider_name']" name="oidc.provider_name" ref="provider_name"
                :disabled="!data['security.oidc']['enabled']" :maxlength="200" />
            </b-field>
          </div>
        </div>

        <div class="columns">
          <div class="column is-6">
            <b-field :label="$t('settings.security.OIDCClientID')" label-position="on-border">
              <b-input v-model="data['security.oidc']['client_id']" name="oidc.client_id" ref="client_id"
                :disabled="!data['security.oidc']['enabled']" :maxlength="200" required />
            </b-field>
          </div>

          <div class="column is-6">
            <b-field :label="$t('settings.security.OIDCClientSecret')" label-position="on-border">
              <b-input v-model="data['security.oidc']['client_secret']" name="oidc.client_secret" type="password"
                :disabled="!data['security.oidc']['enabled']" :maxlength="200" required />
            </b-field>
          </div>
        </div>

        <hr />
        <div class="columns">
          <div class="column is-4">
            <b-field :label="$t('settings.security.OIDCAutoCreateUsers')"
              :message="$t('settings.security.OIDCAutoCreateUsersHelp')">
              <b-switch v-model="data['security.oidc']['auto_create_users']"
                :disabled="!data['security.oidc']['enabled']" name="oidc.auto_create_users" />
            </b-field>
          </div>
          <div class="column is-4">
            <b-field :label="$t('settings.security.OIDCDefaultUserRole')" label-position="on-border"
              :message="$t('settings.security.OIDCDefaultRoleHelp')">
              <b-select v-model="data['security.oidc']['default_user_role_id']"
                :disabled="!data['security.oidc']['enabled'] || !data['security.oidc']['auto_create_users']"
                name="oidc.default_user_role_id" expanded>
                <option v-for="role in userRoles" :key="role.id" :value="role.id">
                  {{ role.name }}
                </option>
              </b-select>
            </b-field>
          </div>
          <div class="column is-4">
            <b-field :label="$t('settings.security.OIDCDefaultListRole')" label-position="on-border"
              :message="$t('settings.security.OIDCDefaultRoleHelp')">
              <b-select v-model="data['security.oidc']['default_list_role_id']"
                :disabled="!data['security.oidc']['enabled'] || !data['security.oidc']['auto_create_users']"
                name="oidc.default_list_role_id" expanded>
                <option :value="null">&mdash; {{ $t("globals.terms.none") }} &mdash;</option>
                <option v-for="role in listRoles" :key="role.id" :value="role.id">
                  {{ role.name }}
                </option>
              </b-select>
            </b-field>
          </div>
        </div>

        <hr />
        <b-field :label="$t('settings.security.OIDCRedirectURL')">
          <code><copy-text :text="`${serverConfig.root_url}/auth/oidc`" /></code>
        </b-field>
        <p v-if="data['security.oidc']['enabled'] && !isURLOk" class="has-text-danger">
          <b-icon icon="warning-empty" />
          {{ $t('settings.security.OIDCRedirectWarning') }}
        </p>
      </div>
    </div>

    <hr />
    <div class="columns">
      <div class="column is-3">
        <b-field :label="$t('settings.security.enableCaptcha')" :message="$t('settings.security.enableCaptchaHelp')">
          <b-switch v-model="captchaEnabled" name="security.captcha" />
        </b-field>
      </div>
      <div class="column is-9" v-if="captchaEnabled">
        <b-field :label="$t('settings.security.captchaProvider')">
          <b-radio v-model="selectedProvider" native-value="altcha" name="captcha_provider">
            ALTCHA
          </b-radio>
          <b-radio v-model="selectedProvider" native-value="hcaptcha" name="captcha_provider">
            hCaptcha (deprecated)
          </b-radio>
        </b-field>

        <!-- captcha settings -->
        <div v-if="selectedProvider === 'altcha'">
          <b-field :label="$t('settings.security.altchaComplexity')" label-position="on-border"
            :message="$t('settings.security.altchaComplexityHelp')">
            <b-input v-model.number="data['security.captcha']['altcha']['complexity']" name="altcha_complexity"
              type="number" min="1000" max="1000000" required />
          </b-field>
        </div>
        <div v-if="selectedProvider === 'hcaptcha'">
          <b-field :label="$t('settings.security.captchaKey')" label-position="on-border"
            :message="$t('settings.security.captchaKeyHelp')">
            <b-input v-model="data['security.captcha']['hcaptcha']['key']" name="hcaptcha_key" :maxlength="200"
              required />
          </b-field>
          <b-field :label="$t('settings.security.captchaSecret')" label-position="on-border">
            <b-input v-model="data['security.captcha']['hcaptcha']['secret']" name="hcaptcha_secret" type="password"
              :maxlength="200" required />
          </b-field>
        </div>
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
    ...mapState(['serverConfig', 'userRoles', 'listRoles']),

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
    this.$api.getUserRoles();
    this.$api.getListRoles();
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
