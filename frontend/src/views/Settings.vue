<template>
  <form @submit.prevent="onSubmit">
    <section class="settings">
      <header class="row page-header">
        <div class="col-8">
          <h1>
            {{ $t('settings.title') }}
            <small>({{ serverConfig.version }})</small>
          </h1>
        </div>
        <div class="col-4 col-end align-right">
          <oat-field v-if="$can('settings:manage')">
            <button :disabled="!hasFormChanged" data-variant="primary" type="submit" class="isSaveEnabled"
              data-cy="btn-save">
              <oat-icon icon="content-save-outline" />
              {{ $t('globals.buttons.save') }}
            </button>
          </oat-field>
        </div>
      </header>

      <div class="card page-content">
        <div v-if="loading.settings || isLoading" aria-busy="true" data-spinner="large overlay" />
        <section class="settings-wrap" v-if="form">
          <ot-tabs class="settings-tabs" @ot-tab-change="tab = $event.detail.index">
            <div role="tablist" aria-orientation="vertical">
              <button v-for="(item, i) in tabs" :key="item.key" type="button" role="tab"
                :aria-selected="tab === i ? 'true' : 'false'">
                {{ item.label }}
              </button>
            </div>

            <div role="tabpanel">
              <general-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <performance-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <privacy-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <security-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <media-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <smtp-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <bounce-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <messenger-settings :form="form" :key="key" />
            </div>

            <div role="tabpanel">
              <appearance-settings :form="form" :key="key" />
            </div>
          </ot-tabs>
        </section>
      </div>
    </section>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import AppearanceSettings from './settings/appearance.vue';
import BounceSettings from './settings/bounces.vue';
import GeneralSettings from './settings/general.vue';
import MediaSettings from './settings/media.vue';
import MessengerSettings from './settings/messengers.vue';
import PerformanceSettings from './settings/performance.vue';
import PrivacySettings from './settings/privacy.vue';
import SecuritySettings from './settings/security.vue';
import SmtpSettings from './settings/smtp.vue';

export default Vue.extend({
  components: {
    GeneralSettings,
    PerformanceSettings,
    PrivacySettings,
    SecuritySettings,
    MediaSettings,
    SmtpSettings,
    BounceSettings,
    MessengerSettings,
    AppearanceSettings,
  },

  data() {
    return {
      // :key="key" is a ack to re-render child components every time settings
      // is pulled. Otherwise, props don't react.
      key: 0,

      isLoading: false,

      // formCopy is a stringified copy of the original settings against which
      // form is compared to detect changes.
      formCopy: '',
      form: null,
      tab: 0,
    };
  },

  methods: {
    async onSubmit() {
      const form = JSON.parse(JSON.stringify(this.form));

      // SMTP cardes.
      let hasDummy = '';
      for (let i = 0; i < form.smtp.length; i += 1) {
        // trim the host before saving
        form.smtp[i].host = form.smtp[i].host?.trim();

        // If it's the dummy UI password placeholder, ignore it.
        if (this.isDummy(form.smtp[i].password)) {
          form.smtp[i].password = '';
        } else if (this.hasDummy(form.smtp[i].password)) {
          hasDummy = `smtp #${i + 1}`;
        }

        if (form.smtp[i].strEmailHeaders && form.smtp[i].strEmailHeaders !== '[]') {
          form.smtp[i].email_headers = JSON.parse(form.smtp[i].strEmailHeaders);
        } else {
          form.smtp[i].email_headers = [];
        }
      }

      // Bounces cardes.
      for (let i = 0; i < form['bounce.mailcardes'].length; i += 1) {
        // trim the host before saving
        form['bounce.mailcardes'][i].host = form['bounce.mailcardes'][i].host?.trim();

        // If it's the dummy UI password placeholder, ignore it.
        if (this.isDummy(form['bounce.mailcardes'][i].password)) {
          form['bounce.mailcardes'][i].password = '';
        } else if (this.hasDummy(form['bounce.mailcardes'][i].password)) {
          hasDummy = `bounce #${i + 1}`;
        }
      }

      if (this.isDummy(form['upload.s3.aws_secret_access_key'])) {
        form['upload.s3.aws_secret_access_key'] = '';
      } else if (this.hasDummy(form['upload.s3.aws_secret_access_key'])) {
        hasDummy = 's3';
      }

      if (this.isDummy(form['bounce.sendgrid_key'])) {
        form['bounce.sendgrid_key'] = '';
      } else if (this.hasDummy(form['bounce.sendgrid_key'])) {
        hasDummy = 'sendgrid';
      }

      if (this.isDummy(form['security.captcha'].hcaptcha.secret)) {
        form['security.captcha'].hcaptcha.secret = '';
      } else if (this.hasDummy(form['security.captcha'].hcaptcha.secret)) {
        hasDummy = 'captcha';
      }

      if (this.isDummy(form['security.oidc'].client_secret)) {
        form['security.oidc'].client_secret = '';
      } else if (this.hasDummy(form['security.oidc'].client_secret)) {
        hasDummy = 'oidc';
      }

      if (this.isDummy(form['bounce.postmark'].password)) {
        form['bounce.postmark'].password = '';
      } else if (this.hasDummy(form['bounce.postmark'].password)) {
        hasDummy = 'postmark';
      }

      if (this.isDummy(form['bounce.forwardemail'].key)) {
        form['bounce.forwardemail'].key = '';
      } else if (this.hasDummy(form['bounce.forwardemail'].key)) {
        hasDummy = 'forwardemail';
      }

      if (this.isDummy(form['bounce.lettermint'].key)) {
        form['bounce.lettermint'].key = '';
      } else if (this.hasDummy(form['bounce.lettermint'].key)) {
        hasDummy = 'lettermint';
      }

      for (let i = 0; i < form.messengers.length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (this.isDummy(form.messengers[i].password)) {
          form.messengers[i].password = '';
        } else if (this.hasDummy(form.messengers[i].password)) {
          hasDummy = `messenger #${i + 1}`;
        }
      }

      if (hasDummy) {
        this.$utils.toast(this.$t('globals.messages.passwordChangeFull', { name: hasDummy }), '');
        return false;
      }

      // Domain blocklist array from multi-line strings.
      form['privacy.domain_blocklist'] = form['privacy.domain_blocklist'].split('\n').map((v) => v.trim().toLowerCase()).filter((v) => v !== '');
      form['privacy.domain_allowlist'] = form['privacy.domain_allowlist'].split('\n').map((v) => v.trim().toLowerCase()).filter((v) => v !== '');

      this.isLoading = true;
      try {
        const data = await this.$api.updateSettings(form);
        await this.$root.awaitRestart(data);
        this.getSettings();
      } finally {
        this.isLoading = false;
      }

      return false;
    },

    getSettings() {
      this.isLoading = true;
      this.$api.getSettings().then((data) => {
        let d = {};
        try {
          // Create a deep-copy of the settings hierarchy.
          d = JSON.parse(JSON.stringify(data));
        } catch (err) {
          return;
        }

        // Serialize the `email_headers` array map to display on the form.
        for (let i = 0; i < d.smtp.length; i += 1) {
          d.smtp[i].strEmailHeaders = JSON.stringify(d.smtp[i].email_headers, null, 4);
        }

        // Domain blocklist array to multi-line string.
        d['privacy.domain_blocklist'] = d['privacy.domain_blocklist'].join('\n');
        d['privacy.domain_allowlist'] = d['privacy.domain_allowlist'].join('\n');

        this.key += 1;
        this.form = d;
        this.formCopy = JSON.stringify(d);

        this.$nextTick(() => {
          this.isLoading = false;
        });
      });
    },

    isDummy(pwd) {
      return !pwd || (pwd.match(/•/g) || []).length === pwd.length;
    },

    hasDummy(pwd) {
      return pwd.includes('•');
    },
  },

  computed: {
    ...mapState(['serverConfig', 'loading']),

    tabs() {
      return [
        { key: 'general', label: this.$t('settings.general.name') },
        { key: 'performance', label: this.$t('settings.performance.name') },
        { key: 'privacy', label: this.$t('settings.privacy.name') },
        { key: 'security', label: this.$t('settings.security.name') },
        { key: 'media', label: this.$t('settings.media.title') },
        { key: 'smtp', label: this.$t('settings.smtp.name') },
        { key: 'bounces', label: this.$t('settings.bounces.name') },
        { key: 'messengers', label: this.$t('settings.messengers.name') },
        { key: 'appearance', label: this.$t('settings.appearance.name') },
      ];
    },

    hasFormChanged() {
      if (!this.formCopy) {
        return false;
      }
      return JSON.stringify(this.form) !== this.formCopy;
    },
  },

  beforeRouteLeave(to, from, next) {
    if (this.hasFormChanged) {
      this.$utils.confirm(this.$t('globals.messages.confirmDiscard'), () => next(true));
      return;
    }
    next(true);
  },

  mounted() {
    this.tab = this.$utils.getPref('settings.tab') || 0;
    this.getSettings();
  },

  watch: {
    tab(t) {
      this.$utils.setPref('settings.tab', t);
    },
  },
});
</script>
