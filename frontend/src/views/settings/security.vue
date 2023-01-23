<template>
  <div class="items">
    <div class="columns">
      <div class="column is-4">
        <b-field :label="$t('settings.security.enableCaptcha')"
          :message="$t('settings.security.enableCaptchaHelp')">
          <b-switch v-model="data['security.enable_captcha']"
              name="security.captcha" />
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
});
</script>
