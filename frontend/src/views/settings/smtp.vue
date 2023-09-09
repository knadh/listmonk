<template>
  <div>
    <div class="items mail-servers">
      <div class="block box" v-for="(item, n) in form.smtp" :key="n">
        <div class="columns">
          <div class="column is-2">
            <b-field :label="$t('globals.buttons.enabled')">
              <b-switch v-model="item.enabled" name="enabled"
                  :native-value="true" data-cy="btn-enable-smtp" />
            </b-field>
            <b-field v-if="form.smtp.length > 1">
              <a @click.prevent="$utils.confirm(null, () => removeSMTP(n))"
                href="#"  data-cy="btn-delete-smtp">
                <b-icon icon="trash-can-outline" />
                {{ $t('globals.buttons.delete') }}
              </a>
            </b-field>
          </div><!-- first column -->

          <div class="column" :class="{'disabled': !item.enabled}">
            <div class="columns">
              <div class="column is-8">
                <b-field :label="$t('settings.mailserver.host')" label-position="on-border"
                  :message="$t('settings.mailserver.hostHelp')">
                  <b-input v-model="item.host" name="host"
                    placeholder='smtp.yourmailserver.net' :maxlength="200" />
                </b-field>
              </div>
              <div class="column">
                <b-field :label="$t('settings.mailserver.port')" label-position="on-border"
                  :message="$t('settings.mailserver.portHelp')">
                  <b-numberinput v-model="item.port" name="port" type="is-light"
                      controls-position="compact"
                      placeholder="25" min="1" max="65535" />
                </b-field>
              </div>
            </div><!-- host -->

            <div class="columns">
              <div class="column is-2">
                <b-field :label="$t('settings.mailserver.authProtocol')"
                  label-position="on-border">
                  <b-select v-model="item.auth_protocol" name="auth_protocol">
                    <option value="login">LOGIN</option>
                    <option value="cram">CRAM</option>
                    <option value="plain">PLAIN</option>
                    <option value="none">None</option>
                  </b-select>
                </b-field>
              </div>
              <div class="column">
                <b-field grouped>
                  <b-field :label="$t('settings.mailserver.username')"
                    label-position="on-border" expanded>
                    <b-input v-model="item.username" :custom-class="`smtp-username-${n}`"
                      :disabled="item.auth_protocol === 'none'"
                      name="username" placeholder="mysmtp" :maxlength="200" />
                  </b-field>
                  <b-field :label="$t('settings.mailserver.password')"
                    label-position="on-border" expanded
                    :message="$t('settings.mailserver.passwordHelp')">
                    <b-input v-model="item.password"
                      :disabled="item.auth_protocol === 'none'"
                      name="password" type="password"
                      :custom-class="`password-${n}`"
                      :placeholder="$t('settings.mailserver.passwordHelp')"
                      :maxlength="200" />
                  </b-field>
                </b-field>
              </div>
            </div><!-- auth -->
            <div class="smtp-shortcuts is-size-7">
              <a href="" @click.prevent="() => fillSettings(n, 'gmail')">Gmail</a>
              <a href="" @click.prevent="() => fillSettings(n, 'ses')">Amazon SES</a>
              <a href="" @click.prevent="() => fillSettings(n, 'mailgun')">Mailgun</a>
              <a href="" @click.prevent="() => fillSettings(n, 'mailjet')">Mailjet</a>
              <a href="" @click.prevent="() => fillSettings(n, 'sendgrid')">Sendgrid</a>
              <a href="" @click.prevent="() => fillSettings(n, 'postmark')">Postmark</a>
            </div>
            <hr />

            <div class="columns">
              <div class="column is-6">
                <b-field :label="$t('settings.smtp.heloHost')" label-position="on-border"
                  :message="$t('settings.smtp.heloHostHelp')">
                  <b-input v-model="item.hello_hostname"
                    name="hello_hostname" placeholder="" :maxlength="200" />
                </b-field>
              </div>
              <div class="column">
                <b-field grouped>
                  <b-field :label="$t('settings.mailserver.tls')" expanded
                    :message="$t('settings.mailserver.tlsHelp')" label-position="on-border">
                    <b-select v-model="item.tls_type" name="items.tls_type">
                      <option value="none">{{ $t('globals.states.off') }}</option>
                      <option value="STARTTLS">STARTTLS</option>
                      <option value="TLS">SSL/TLS</option>
                    </b-select>
                  </b-field>
                  <b-field :label="$t('settings.mailserver.skipTLS')" expanded
                    :message="$t('settings.mailserver.skipTLSHelp')">
                    <b-switch v-model="item.tls_skip_verify"
                      :disabled="item.tls_type === 'none'" name="item.tls_skip_verify" />
                  </b-field>
                </b-field>
              </div>
            </div><!-- TLS -->
            <hr />

            <div class="columns">
              <div class="column is-3">
                <b-field :label="$t('settings.mailserver.maxConns')"
                  label-position="on-border"
                  :message="$t('settings.mailserver.maxConnsHelp')">
                  <b-numberinput v-model="item.max_conns" name="max_conns" type="is-light"
                      controls-position="compact"
                      placeholder="25" min="1" max="65535" />
                </b-field>
              </div>
              <div class="column is-3">
                <b-field :label="$t('settings.smtp.retries')" label-position="on-border"
                  :message="$t('settings.smtp.retriesHelp')">
                  <b-numberinput v-model="item.max_msg_retries" name="max_msg_retries"
                      type="is-light"
                      controls-position="compact"
                      placeholder="2" min="1" max="1000" />
                </b-field>
              </div>
              <div class="column is-3">
                <b-field :label="$t('settings.mailserver.idleTimeout')"
                  label-position="on-border"
                  :message="$t('settings.mailserver.idleTimeoutHelp')">
                  <b-input v-model="item.idle_timeout" name="idle_timeout"
                    placeholder="15s" :pattern="regDuration" :maxlength="10" />
                </b-field>
              </div>
              <div class="column is-3">
                <b-field :label="$t('settings.mailserver.waitTimeout')"
                  label-position="on-border"
                  :message="$t('settings.mailserver.waitTimeoutHelp')">
                  <b-input v-model="item.wait_timeout" name="wait_timeout"
                    placeholder="5s" :pattern="regDuration" :maxlength="10" />
                </b-field>
              </div>
            </div>

            <div class="columns">
              <div class="column">
                <p v-if="item.email_headers.length === 0 && !item.showHeaders">
                  <a href="#" @click.prevent="() => showSMTPHeaders(n)">
                    <b-icon icon="plus" />{{ $t('settings.smtp.setCustomHeaders') }}</a>
                </p>
                <b-field v-if="item.email_headers.length > 0 || item.showHeaders"
                  label-position="on-border"
                  :message="$t('settings.smtp.customHeadersHelp')">
                  <b-input v-model="item.strEmailHeaders" name="email_headers" type="textarea"
                    placeholder='[{"X-Custom": "value"}, {"X-Custom2": "value"}]' />
                </b-field>
              </div>
            </div>
            <hr />

            <form @submit.prevent="() => doSMTPTest(item, n)">
              <div class="columns">
                <template v-if="smtpTestItem === n">
                  <div class="column is-5">
                    <strong>{{ $t('settings.general.fromEmail') }}</strong>
                    <br />
                    {{ settings['app.from_email'] }}
                  </div>
                  <div class="column is-4">
                    <b-field :label="$t('settings.smtp.toEmail')" label-position="on-border">
                      <b-input type="email" required v-model="testEmail"
                        :ref="'testEmailTo'" placeholder="email@site.com"
                        :custom-class="`test-email-${n}`" />
                    </b-field>
                  </div>
                </template>
                <div class="column has-text-right">
                  <b-button v-if="smtpTestItem === n" class="is-primary"
                    @click.prevent="() => doSMTPTest(item, n)">
                    {{ $t('settings.smtp.sendTest') }}
                  </b-button>
                  <a href="#" v-else class="is-primary" @click.prevent="showTestForm(n)">
                    <b-icon icon="rocket-launch-outline" /> {{ $t('settings.smtp.testConnection') }}
                  </a>
                </div>
                <div class="columns">
                  <div class="column">
                  </div>
                </div>
              </div>
              <div v-if="errMsg && smtpTestItem === n">
                <b-field class="mt-4" type="is-danger">
                  <b-input v-model="errMsg" type="textarea"
                    custom-class="has-text-danger is-size-6" readonly />
                </b-field>
              </div>
            </form><!-- smtp test -->

          </div>
        </div><!-- second container column -->
      </div><!-- block -->
    </div><!-- mail-servers -->

    <b-button @click="addSMTP" icon-left="plus" type="is-primary">
      {{ $t('globals.buttons.addNew') }}
    </b-button>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import { regDuration } from '../../constants';

const smtpTemplates = {
  gmail: {
    host: 'smtp.gmail.com', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  ses: {
    host: 'email-smtp.YOUR-REGION.amazonaws.com', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  mailjet: {
    host: 'in-v3.mailjet.com', port: 465, auth_protocol: 'cram', tls_type: 'TLS',
  },
  mailgun: {
    host: 'smtp.mailgun.org', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  sendgrid: {
    host: 'smtp.sendgrid.net', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  postmark: {
    host: 'smtp.postmarkapp.com', port: 587, auth_protocol: 'cram', tls_type: 'STARTTLS',
  },
};

export default Vue.extend({
  props: {
    form: {
      type: Object,
    },
  },

  data() {
    return {
      data: this.form,
      regDuration,
      // Index of the SMTP block item in the array to show the
      // test form in.
      smtpTestItem: null,
      testEmail: '',
      errMsg: '',
    };
  },

  methods: {
    addSMTP() {
      this.data.smtp.push({
        enabled: true,
        host: '',
        hello_hostname: '',
        port: 587,
        auth_protocol: 'none',
        username: '',
        password: '',
        email_headers: [],
        max_conns: 10,
        max_msg_retries: 2,
        idle_timeout: '15s',
        wait_timeout: '5s',
        tls_type: 'STARTTLS',
        tls_skip_verify: false,
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.mail-servers input[name="host"]');
        items[items.length - 1].focus();
      });
    },

    removeSMTP(i) {
      this.data.smtp.splice(i, 1);
    },

    showSMTPHeaders(i) {
      const s = this.data.smtp[i];
      s.showHeaders = true;
      this.data.smtp.splice(i, 1, s);
    },

    testConnection() {
      let em = this.settings['app.from_email'].replace('>', '').split('<');
      if (em.length > 1) {
        em = `<${em[em.length - 1]}>`;
      }
    },

    doSMTPTest(item, n) {
      if (!this.isTestEnabled(item)) {
        this.$utils.toast(this.$t('settings.smtp.testEnterEmail'), 'is-danger');
        this.$nextTick(() => {
          const i = document.querySelector(`.password-${n}`);
          this.data.smtp[n].password = '';
          i.focus();
          i.select();
        });
        return;
      }

      this.errMsg = '';
      this.$api.testSMTP({ ...item, email: this.testEmail }).then(() => {
        this.$utils.toast(this.$t('campaigns.testSent'));
      }).catch((err) => {
        if (err.response?.data?.message) {
          this.errMsg = err.response.data.message;
        }
      });
    },

    showTestForm(n) {
      this.smtpTestItem = n;
      this.testItem = this.form.smtp[n];
      this.errMsg = '';

      this.$nextTick(() => {
        document.querySelector(`.test-email-${n}`).focus();
      });
    },

    isTestEnabled(item) {
      if (!item.host || !item.port) {
        return false;
      }
      if (item.auth_protocol !== 'none' && item.password.includes('•')) {
        return false;
      }

      return true;
    },

    fillSettings(n, key) {
      this.data.smtp.splice(n, 1, {
        ...this.data.smtp[n],
        ...smtpTemplates[key],
        username: '',
        password: '',
        hello_hostname: '',
        tls_skip_verify: false,
      });

      this.$nextTick(() => {
        document.querySelector(`.smtp-username-${n}`).focus();
      });
    },
  },

  computed: {
    ...mapState(['settings']),
  },
});
</script>
