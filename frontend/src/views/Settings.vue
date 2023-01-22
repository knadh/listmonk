<template>
  <form @submit.prevent="onSubmit">
    <section class="settings">
      <b-loading :is-full-page="true" v-if="loading.settings || isLoading" active />
      <header class="columns page-header">
        <div class="column is-half">
          <h1 class="title is-4">{{ $t('settings.title') }}
            <span class="has-text-grey-light">({{ serverConfig.version }})</span>
          </h1>
        </div>
        <div class="column has-text-right">
          <b-field expanded>
            <b-button expanded :disabled="!hasFormChanged"
              type="is-primary" icon-left="content-save-outline" native-type="submit"
              class="isSaveEnabled" data-cy="btn-save">
              {{ $t('globals.buttons.save') }}
            </b-button>
          </b-field>
        </div>
      </header>
      <hr />

      <section class="wrap">
          <b-tabs type="is-boxed" :animated="false" v-model="tab">
            <b-tab-item :label="$t('settings.general.name')" label-position="on-border">
              <general-settings :form="form" :key="key" />
            </b-tab-item><!-- general -->

            <b-tab-item :label="$t('settings.performance.name')">
              <performance-settings :form="form" :key="key" />
            </b-tab-item><!-- performance -->

            <b-tab-item :label="$t('settings.privacy.name')">
              <privacy-settings :form="form" :key="key" />
            </b-tab-item><!-- privacy -->

            <b-tab-item :label="$t('settings.security.name')">
              <security-settings :form="form" :key="key" />
            </b-tab-item><!-- security -->

            <b-tab-item :label="$t('settings.media.title')">
              <media-settings :form="form" :key="key" />
            </b-tab-item><!-- media -->

            <b-tab-item :label="$t('settings.smtp.name')">
              <smtp-settings :form="form" :key="key" />
            </b-tab-item><!-- mail servers -->

            <b-tab-item :label="$t('settings.bounces.name')">
              <bounce-settings :form="form" :key="key" />
            </b-tab-item><!-- bounces -->

            <b-tab-item :label="$t('settings.messengers.name')">
              <messenger-settings :form="form" :key="key" />
            </b-tab-item><!-- messengers -->

            <b-tab-item :label="$t('settings.appearance.name')">
              <appearance-settings :form="form" :key="key" />
            </b-tab-item><!-- appearance -->
          </b-tabs>

      </section>
    </section>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import GeneralSettings from './settings/general.vue';
import PerformanceSettings from './settings/performance.vue';
import PrivacySettings from './settings/privacy.vue';
import SecuritySettings from './settings/security.vue';
import MediaSettings from './settings/media.vue';
import SmtpSettings from './settings/smtp.vue';
import BounceSettings from './settings/bounces.vue';
import MessengerSettings from './settings/messengers.vue';
import AppearanceSettings from './settings/appearance.vue';

const dummyPassword = ' '.repeat(8);

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
      form: {},
      tab: 0,
    };
  },

  methods: {
    onSubmit() {
      const form = JSON.parse(JSON.stringify(this.form));

      // SMTP boxes.
      for (let i = 0; i < form.smtp.length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form.smtp[i].password === dummyPassword) {
          form.smtp[i].password = '';
        }

        if (form.smtp[i].strEmailHeaders && form.smtp[i].strEmailHeaders !== '[]') {
          form.smtp[i].email_headers = JSON.parse(form.smtp[i].strEmailHeaders);
        } else {
          form.smtp[i].email_headers = [];
        }
      }

      // Bounces boxes.
      for (let i = 0; i < form['bounce.mailboxes'].length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form['bounce.mailboxes'][i].password === dummyPassword) {
          form['bounce.mailboxes'][i].password = '';
        }
      }

      if (form['upload.s3.aws_secret_access_key'] === dummyPassword) {
        form['upload.s3.aws_secret_access_key'] = '';
      }

      if (form['bounce.sendgrid_key'] === dummyPassword) {
        form['bounce.sendgrid_key'] = '';
      }

      if (form['security.captcha_secret'] === dummyPassword) {
        form['security.captcha_secret'] = '';
      }

      for (let i = 0; i < form.messengers.length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form.messengers[i].password === dummyPassword) {
          form.messengers[i].password = '';
        }
      }

      // Domain blocklist array from multi-line strings.
      form['privacy.domain_blocklist'] = form['privacy.domain_blocklist'].split('\n').map((v) => v.trim().toLowerCase()).filter((v) => v !== '');

      this.isLoading = true;
      this.$api.updateSettings(form).then((data) => {
        if (data.needsRestart) {
          // There are running campaigns and the app didn't auto restart.
          // The UI will show a warning.
          this.$root.loadConfig();
          this.getSettings();
          this.isLoading = false;
          return;
        }

        this.$utils.toast(this.$t('settings.messengers.messageSaved'));

        // Poll until there's a 200 response, waiting for the app
        // to restart and come back up.
        const pollId = setInterval(() => {
          this.$api.getHealth().then(() => {
            clearInterval(pollId);
            this.$root.loadConfig();
            this.getSettings();
          });
        }, 500);
      }, () => {
        this.isLoading = false;
      });
    },

    getSettings() {
      this.isLoading = true;
      this.$api.getSettings().then((data) => {
        const d = JSON.parse(JSON.stringify(data));

        // Serialize the `email_headers` array map to display on the form.
        for (let i = 0; i < d.smtp.length; i += 1) {
          d.smtp[i].strEmailHeaders = JSON.stringify(d.smtp[i].email_headers, null, 4);

          // The backend doesn't send passwords, so add a dummy so that
          // the password looks filled on the UI.
          d.smtp[i].password = dummyPassword;
        }

        for (let i = 0; i < d['bounce.mailboxes'].length; i += 1) {
          // The backend doesn't send passwords, so add a dummy so that
          // the password looks filled on the UI.
          d['bounce.mailboxes'][i].password = dummyPassword;
        }

        for (let i = 0; i < d.messengers.length; i += 1) {
          // The backend doesn't send passwords, so add a dummy so that it
          // the password looks filled on the UI.
          d.messengers[i].password = dummyPassword;
        }

        if (d['upload.provider'] === 's3') {
          d['upload.s3.aws_secret_access_key'] = dummyPassword;
        }
        d['bounce.sendgrid_key'] = dummyPassword;
        d['security.captcha_secret'] = dummyPassword;

        // Domain blocklist array to multi-line string.
        d['privacy.domain_blocklist'] = d['privacy.domain_blocklist'].join('\n');

        this.key += 1;
        this.form = d;
        this.formCopy = JSON.stringify(d);

        this.$nextTick(() => {
          this.isLoading = false;
        });
      });
    },
  },

  computed: {
    ...mapState(['serverConfig', 'loading']),

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
