<template>
  <section class="user-profile">
    <header class="row page-header">
      <div class="col-8">
        <h1>
          @{{ data.username }}
        </h1>
        <span v-if="data.userRole" class="badge">{{ data.userRole.name }}</span>
      </div>
    </header>

    <div class="card page-content">
    <oat-loading v-if="loading.users" :active="loading.users" :is-full-page="false" />
    <form @submit.prevent="onSubmit">
      <oat-field v-if="data.type !== 'api'" :label="$t('subscribers.email')">
        <input aria-label="field" :maxlength="200" v-model="form.email" name="email" :placeholder="$t('subscribers.email')"
          :disabled="!data.passwordLogin" required>
      </oat-field>

      <oat-field :label="$t('globals.fields.name')">
        <input aria-label="field" :maxlength="200" v-model="form.name" name="name" :placeholder="$t('globals.fields.name')">
      </oat-field>

      <div v-if="data.passwordLogin" class="row">
        <div class="col-6">
          <oat-field :label="$t('users.password')">
            <input aria-label="field" minlength="8" :maxlength="200" v-model="form.password" type="password" name="password"
              :placeholder="$t('users.password')">
          </oat-field>
        </div>
        <div class="col-6">
          <oat-field :label="$t('users.passwordRepeat')">
            <input aria-label="field" minlength="8" :maxlength="200" v-model="form.password2" type="password" name="password2">
          </oat-field>
        </div>
      </div>

      <oat-field>
        <button data-variant="primary" type="submit" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </button>
      </oat-field>
    </form>

    <br /><br />

    <!-- 2FA -->
    <section v-if="this.data.passwordLogin" class="twofa-app-section">
      <!-- TOTP disabled -->
      <div v-if="data.twofaType === 'none'" class="card">
        <div class="row mb-4">
          <div class="col-12">
            <h3 class="mb-0">{{ $t('users.twoFA') }}</h3>
          </div>
          <div class="col-2">
            <oat-switch v-if="!isTotpVisible" v-model="twofaEnabled" @input="onToggleEnableTotp" />
          </div>
        </div>

        <p>{{ $t('users.twoFANotEnabled') }}</p>
        <br />

        <!-- TOTP setup -->
        <div v-if="isTotpVisible" class="totp-setup">
          <div v-if="totpQR" class="qr-app-section">
            <p class="text-light text-7">{{ $t('users.totpScanQR') }}</p><br />

            <img :src="'data:image/png;base64,' + totpQR" alt="QR Code" />

            <br /><br />
            <p>
              <strong>{{ $t('users.totpSecret') }}</strong><br />
              <code><copy-text :text="`${totpSecret}`" /></code>
            </p>

            <br /><br />
            <form @submit.prevent="confirmTOTP">
              <oat-field :label="$t('users.totpCode')">
                <input aria-label="field" ref="totpCodeInput" v-model="totpCode" maxlength="6" pattern="[0-9]{6}" placeholder="000000"
                  required>
              </oat-field>
              <div class="hstack">
                <button data-variant="primary" type="submit">
                  {{ $t('globals.buttons.enable') }}
                </button>
                <button type="button" @click="onCancelTOTPSetup">
                  {{ $t('globals.buttons.cancel') }}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>

      <!-- TOTP Enabled -->
      <div v-if="data.twofaType === 'totp'" class="card">
        <div class="row">
          <div class="col-12">
            <h3>
              <oat-icon icon="check-circle-outline" data-variant="primary" /> {{ $t('users.twoFAEnabled') }}
            </h3>
          </div>
          <div class="col-2">
            <oat-switch v-if="!showDisableTOTP" v-model="twofaEnabled" @input="toggleDisableTOTP" />
          </div>
        </div>

        <p>{{ $t('users.twoFAEnabledDesc', { type: data.twofaType.toUpperCase() }) }}</p>

        <!-- Disable TOTP Flow -->
        <form v-if="showDisableTOTP" class="disable-totp mt-5" @submit.prevent="confirmDisableTOTP">
          <oat-field :label="$t('users.password')">
            <input aria-label="field" ref="disablePasswordInput" v-model="disableTOTPPassword" type="password" minlength="8" required>
          </oat-field>
          <div class="hstack">
            <button data-variant="danger" type="submit">
              {{ $t('globals.buttons.disable') }}
            </button>
            <button type="button" @click="onCancelTOTPSetup">
              {{ $t('globals.buttons.cancel') }}
            </button>
          </div>
        </form>
      </div>
    </section>
    </div>
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
      isTotpVisible: false,
      totpQR: null,
      totpSecret: null,
      totpCode: '',
      showDisableTOTP: false,
      disableTOTPPassword: '',
      twofaEnabled: false,
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
          this.$utils.toast(this.$t('users.passwordMismatch'), '');
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

    onToggleEnableTotp() {
      this.$api.getTOTPQR(this.data.id).then((data) => {
        this.totpQR = data.qr;
        this.totpSecret = data.secret;
        this.isTotpVisible = true;

        this.$nextTick(() => {
          if (this.$refs.totpCodeInput) {
            this.$refs.totpCodeInput.focus();
          }
        });
      }).catch(() => {
        this.$utils.toast(this.$t('globals.messages.errorFetching'), '');
      });
    },

    onCancelTOTPSetup() {
      this.isTotpVisible = false;
      this.totpQR = null;
      this.totpSecret = null;
      this.totpCode = '';
      this.twofaEnabled = this.data.twofaType === 'totp';
      this.showDisableTOTP = false;
      this.disableTOTPPassword = '';
    },

    confirmTOTP() {
      if (!this.totpCode || this.totpCode.length !== 6) {
        this.$utils.toast(this.$t('globals.messages.invalidValue'), '');
        return;
      }

      const d = new FormData();
      d.append('secret', this.totpSecret);
      d.append('code', this.totpCode);

      this.$api.enableTOTP(this.data.id, d).then(() => {
        this.$utils.toast(this.$t('users.twoFAEnabled'));
        this.onCancelTOTPSetup();

        // Reload user profile
        this.$api.getUserProfile().then((data) => {
          this.data = { ...data };
          this.twofaEnabled = data.twofaType === 'totp';
        });
      }).catch(() => {
        this.$utils.toast(this.$t('globals.messages.invalidValue'), '');
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
        this.$utils.toast(this.$t('globals.messages.invalidFields'), '');
        return;
      }

      const formData = new FormData();
      formData.append('password', this.disableTOTPPassword);

      this.$api.disableTOTP(this.data.id, formData).then(() => {
        this.$utils.toast(this.$t('globals.messages.done'));
        this.showDisableTOTP = false;
        this.disableTOTPPassword = '';
        // Reload user profile
        this.$api.getUserProfile().then((data) => {
          this.data = { ...data };
          this.twofaEnabled = data.twofaType === 'totp';
        });
      }).catch(() => {
        this.$utils.toast(this.$t('users.invalidPassword'), '');
      });
    },
  },

  mounted() {
    this.$api.getUserProfile().then((data) => {
      this.data = { ...data };
      this.form = { name: data.name, email: data.email };
      this.twofaEnabled = data.twofaType === 'totp';
    });
  },

  computed: {
    ...mapState(['loading']),
  },

});
</script>
