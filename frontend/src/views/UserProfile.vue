<template>
  <section class="user-profile section-mini">
    <b-loading v-if="loading.users" :active="loading.users" :is-full-page="false" />

    <h1 class="title">
      @{{ data.username }}
    </h1>
    <b-tag v-if="data.userRole">{{ data.userRole.name }}</b-tag>

    <br /><br /><br />
    <form @submit.prevent="onSubmit">
      <b-field v-if="data.type !== 'api'" :label="$t('subscribers.email')" label-position="on-border">
        <b-input :maxlength="200" v-model="form.email" name="email" :placeholder="$t('subscribers.email')"
          :disabled="!data.passwordLogin" required autofocus />
      </b-field>

      <b-field :label="$t('globals.fields.name')" label-position="on-border">
        <b-input :maxlength="200" v-model="form.name" name="name" :placeholder="$t('globals.fields.name')" />
      </b-field>

      <div v-if="data.passwordLogin" class="columns">
        <div class="column is-6">
          <b-field :label="$t('users.password')" label-position="on-border">
            <b-input minlength="8" :maxlength="200" v-model="form.password" type="password" name="password"
              :placeholder="$t('users.password')" />
          </b-field>
        </div>
        <div class="column is-6">
          <b-field :label="$t('users.passwordRepeat')" label-position="on-border">
            <b-input minlength="8" :maxlength="200" v-model="form.password2" type="password" name="password2" />
          </b-field>
        </div>
      </div>

      <b-field expanded>
        <b-button type="is-primary" icon-left="content-save-outline" native-type="submit" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </b-field>
    </form>

    <br /><br />

    <!-- 2FA Section -->
    <section class="twofa-section">
      <!-- TOTP Not Enabled -->
      <div v-if="data.twofaType === 'none'" class="box">
        <h3 class="title is-size-5 mb-4">{{ $t('users.twoFA') }}</h3>

        <p>{{ $t('users.twoFANotEnabled') }}</p>
        <br />
        <b-button v-if="!showTOTPSetup" type="is-primary" @click="toggleEnableTOTP">
          {{ $t('users.enableTOTP') }}
        </b-button>

        <!-- TOTP Setup Flow -->
        <div v-if="showTOTPSetup" class="totp-setup">
          <div v-if="totpQR" class="qr-section">
            <p class="has-text-grey">{{ $t('users.totpScanQR') }}</p><br />

            <img :src="'data:image/png;base64,' + totpQR" alt="QR Code" />

            <br /><br />
            <p>
              <strong>{{ $t('users.totpSecret') }}</strong><br />
              <code><copy-text :text="`${totpSecret}`" /></code>
            </p>

            <br /><br />
            <form @submit.prevent="confirmTOTP">
              <b-field :label="$t('users.totpCode')" label-position="on-border">
                <b-input ref="totpCodeInput" v-model="totpCode" maxlength="6" pattern="[0-9]{6}" placeholder="000000"
                  required />
              </b-field>
              <div class="buttons">
                <b-button type="is-primary" native-type="submit">
                  {{ $t('globals.buttons.continue') }}
                </b-button>
                <b-button type="button" @click="cancelTOTPSetup">
                  {{ $t('globals.buttons.cancel') }}
                </b-button>
              </div>
            </form>
          </div>
        </div>
      </div>

      <!-- TOTP Enabled -->
      <div v-if="data.twofaType === 'totp'" class="box">
        <h3 class="title is-size-5 mb-4">
          <b-icon icon="check-circle-outline" type="is-success" /> {{ $t('users.twoFAEnabled') }}
        </h3>
        <p>{{ $t('users.twoFAEnabledDesc', { type: data.twofaType.toUpperCase() }) }}</p>
        <br />
        <b-button v-if="!showDisableTOTP" type="is-danger" @click="toggleDisableTOTP">
          {{ $t('users.disableTOTP') }}
        </b-button>

        <!-- Disable TOTP Flow -->
        <div v-if="showDisableTOTP" class="disable-totp">
          <form @submit.prevent="confirmDisableTOTP">
            <b-field :label="$t('users.password')" label-position="on-border">
              <b-input ref="disablePasswordInput" v-model="disableTOTPPassword" type="password" minlength="8"
                required />
            </b-field>
            <div class="buttons">
              <b-button type="is-danger" native-type="submit">
                {{ $t('globals.buttons.continue') }}
              </b-button>
              <b-button type="button" @click="cancelDisableTOTP">
                {{ $t('globals.buttons.cancel') }}
              </b-button>
            </div>
          </form>
        </div>
      </div>
    </section>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CopyText from '../components/CopyText.vue';

export default Vue.extend({
  name: 'UserProfile',

  components: {
    CopyText,
  },

  data() {
    return {
      form: {},
      data: {},
      showTOTPSetup: false,
      totpQR: null,
      totpSecret: null,
      totpCode: '',
      showDisableTOTP: false,
      disableTOTPPassword: '',
    };
  },

  methods: {
    onSubmit() {
      const params = {
        name: this.form.name,
        email: this.form.email,
      };

      if (this.data.passwordLogin && this.form.password) {
        if (this.form.password !== this.form.password2) {
          this.$utils.toast(this.$t('users.passwordMismatch'), 'is-danger');
          return;
        }

        params.password = this.form.password;
        params.password2 = this.form.password2;
      }

      this.$api.updateUserProfile(params).then(() => {
        this.form.password = '';
        this.form.password2 = '';
        this.$utils.toast(this.$t('globals.messages.updated', { name: this.data.username }));
      });
    },

    toggleEnableTOTP() {
      this.initTOTPSetup();
    },

    initTOTPSetup() {
      this.$api.getTOTPQR(this.data.id).then((data) => {
        this.totpQR = data.qr;
        this.totpSecret = data.secret;
        this.showTOTPSetup = true;

        this.$nextTick(() => {
          if (this.$refs.totpCodeInput) {
            this.$refs.totpCodeInput.focus();
          }
        });
      }).catch(() => {
        this.$utils.toast(this.$t('globals.messages.errorFetching'), 'is-danger');
      });
    },

    cancelTOTPSetup() {
      this.showTOTPSetup = false;
      this.totpQR = null;
      this.totpSecret = null;
      this.totpCode = '';
    },

    confirmTOTP() {
      if (!this.totpCode || this.totpCode.length !== 6) {
        this.$utils.toast(this.$t('users.invalidTOTPCode'), 'is-danger');
        return;
      }

      const formData = new FormData();
      formData.append('secret', this.totpSecret);
      formData.append('code', this.totpCode);

      this.$api.enableTOTP(this.data.id, formData).then(() => {
        this.$utils.toast(this.$t('users.twoFAEnabled'));
        this.cancelTOTPSetup();
        // Reload user profile
        this.$api.getUserProfile().then((data) => {
          this.data = { ...data };
        });
      }).catch(() => {
        this.$utils.toast(this.$t('users.invalidTOTPCode'), 'is-danger');
      });
    },

    toggleDisableTOTP() {
      this.showDisableTOTP = true;

      this.$nextTick(() => {
        if (this.$refs.disablePasswordInput) {
          this.$refs.disablePasswordInput.focus();
        }
      });
    },

    cancelDisableTOTP() {
      this.showDisableTOTP = false;
      this.disableTOTPPassword = '';
    },

    confirmDisableTOTP() {
      if (!this.disableTOTPPassword) {
        this.$utils.toast(this.$t('globals.messages.invalidFields'), 'is-danger');
        return;
      }

      const formData = new FormData();
      formData.append('password', this.disableTOTPPassword);

      this.$api.disableTOTP(this.data.id, formData).then(() => {
        this.$utils.toast(this.$t('users.twoFADisabled'));
        this.showDisableTOTP = false;
        this.disableTOTPPassword = '';
        // Reload user profile
        this.$api.getUserProfile().then((data) => {
          this.data = { ...data };
        });
      }).catch(() => {
        this.$utils.toast(this.$t('users.invalidPassword'), 'is-danger');
      });
    },
  },

  mounted() {
    this.$api.getUserProfile().then((data) => {
      this.data = { ...data };
      this.form = { name: data.name, email: data.email };
    });
  },

  computed: {
    ...mapState(['loading']),
  },

});
</script>
