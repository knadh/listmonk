<template>
  <div>
    <div class="items mail-servers">
      <div class="card" v-for="(item, n) in form.smtp" :key="n">
        <div class="row">
          <div class="col-2">
            <oat-field>
              <oat-switch v-model="item.enabled" name="enabled" :native-value="true" data-cy="btn-enable-smtp">
                {{ $t('globals.buttons.enabled') }}
              </oat-switch>
            </oat-field>
            <oat-field v-if="form.smtp.length > 1">
              <a @click.prevent="$utils.confirm(null, () => removeSMTP(n))" href="#" data-cy="btn-delete-smtp">
                <oat-icon icon="trash-can-outline" />
                {{ $t('globals.buttons.delete') }}
              </a>
            </oat-field>
          </div><!-- first col-12 -->

          <div class="col-10" :class="{ disabled: !item.enabled }">
            <div class="row">
              <div class="col-9">
                <oat-field :label="$t('settings.mailserver.host')"
                  :message="$t('settings.mailserver.hostHelp')">
                  <input aria-label="field" v-model="item.host" name="host" placeholder="smtp.yourmailserver.net" :maxlength="200">
                </oat-field>
              </div>
              <div class="col-3">
                <oat-field :label="$t('settings.mailserver.port')"
                  :message="$t('settings.mailserver.portHelp')">
                  <input aria-label="field" type="number" v-model.number="item.port" name="port"
                    placeholder="25" min="1" max="65535">
                </oat-field>
              </div>
            </div><!-- host -->

            <div class="row">
              <div class="col-3">
                <oat-field :label="$t('settings.mailserver.authProtocol')">
                  <select aria-label="field" v-model="item.auth_protocol" name="auth_protocol">
                    <option value="login">
                      LOGIN
                    </option>
                    <option value="cram">
                      CRAM
                    </option>
                    <option value="plain">
                      PLAIN
                    </option>
                    <option value="none">
                      None
                    </option>
                  </select>
                </oat-field>
              </div>
              <div class="col-9">
                <oat-field>
                  <oat-field :label="$t('settings.mailserver.username')">
                    <input aria-label="field" v-model="item.username"
                      :disabled="item.auth_protocol === 'none'" name="username" placeholder="mysmtp" :maxlength="200">
                  </oat-field>
                  <oat-field :label="$t('settings.mailserver.password')"
                    :message="$t('settings.mailserver.passwordHelp')">
                    <input aria-label="field" v-model="item.password" :disabled="item.auth_protocol === 'none'" name="password"
                      type="password"
                      :placeholder="$t('settings.mailserver.passwordHelp')" :maxlength="200">
                  </oat-field>
                </oat-field>
              </div>
            </div><!-- auth -->
            <div class="spaced-links">
              <a href="#" @click.prevent="() => fillSettings(n, 'gmail')">Gmail</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'ses')">Amazon SES</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'azure')">Azure ACS</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'mailgun')">Mailgun</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'mailjet')">Mailjet</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'sendgrid')">Sendgrid</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'postmark')">Postmark</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'forwardemail')">Forward Email</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'lettermint')">Lettermint</a>
            </div>
            <hr />

            <div class="row">
              <div class="col-6">
                <oat-field :label="$t('settings.smtp.heloHost')"
                  :message="$t('settings.smtp.heloHostHelp')">
                  <input aria-label="field" v-model="item.hello_hostname" name="hello_hostname" placeholder="" :maxlength="200">
                </oat-field>
              </div>
              <div class="col-6">
                <oat-field>
                  <oat-field :label="$t('settings.mailserver.tls')" :message="$t('settings.mailserver.tlsHelp')"
                   >
                    <select aria-label="field" v-model="item.tls_type" name="items.tls_type">
                      <option value="none">
                        {{ $t('globals.states.off') }}
                      </option>
                      <option value="STARTTLS">
                        STARTTLS
                      </option>
                      <option value="TLS">
                        SSL/TLS
                      </option>
                    </select>
                  </oat-field>
                  <oat-field :message="$t('settings.mailserver.skipTLSHelp')">
                    <oat-switch v-model="item.tls_skip_verify" :disabled="item.tls_type === 'none'"
                      name="item.tls_skip_verify">
                      {{ $t('settings.mailserver.skipTLS') }}
                    </oat-switch>
                  </oat-field>
                </oat-field>
              </div>
            </div><!-- TLS -->
            <hr />

            <div class="row">
              <div class="col-4">
                <oat-field :label="$t('settings.mailserver.maxConns')"
                  :message="$t('settings.mailserver.maxConnsHelp')">
                  <input aria-label="field" type="number" v-model.number="item.max_conns" name="max_conns"
                    placeholder="25" min="1" max="65535">
                </oat-field>
              </div>
              <div class="col-4">
                <oat-field :label="$t('settings.mailserver.idleTimeout')"
                  :message="$t('settings.mailserver.idleTimeoutHelp')">
                  <input aria-label="field" v-model="item.idle_timeout" name="idle_timeout" placeholder="15s" :pattern="regDuration"
                    :maxlength="10">
                </oat-field>
              </div>
              <div class="col-4">
                <oat-field :label="$t('settings.mailserver.waitTimeout')"
                  :message="$t('settings.mailserver.waitTimeoutHelp')">
                  <input aria-label="field" v-model="item.wait_timeout" name="wait_timeout" placeholder="5s" :pattern="regDuration"
                    :maxlength="10">
                </oat-field>
              </div>
            </div>

            <div class="row">
              <div class="col-4">
                <oat-field :label="$t('settings.smtp.retries')"
                  :message="$t('settings.smtp.retriesHelp')">
                  <input aria-label="field" type="number" v-model.number="item.max_msg_retries" name="max_msg_retries" placeholder="2" min="1" max="1000">
                </oat-field>
              </div>
              <div class="col-4">
                <oat-field :label="$t('settings.smtp.retryDelay')"
                  :message="$t('settings.smtp.retryDelayHelp')">
                  <input aria-label="field" v-model="item.msg_retry_delay" name="msg_retry_delay" placeholder="0s" :pattern="regDuration"
                    :maxlength="10">
                </oat-field>
              </div>
            </div>

            <hr />
            <div class="row">
              <div class="col-6">
                <oat-field :label="$t('globals.fields.name')"
                  :message="$t('settings.mailserver.nameHelp')">
                  <input aria-label="field" v-model="item.name" name="name" placeholder="email-primary" :maxlength="100">
                </oat-field>
              </div>
              <div class="col-6">
                <oat-field :label="$t('settings.smtp.fromAddresses')"
                  :message="$t('settings.smtp.fromAddressesHelp')">
                  <oat-tag-input v-model="item.from_addresses" name="from_addresses"
                    :before-adding="validateFromAddress" placeholder="user@example.com, anothersite.com" />
                </oat-field>
              </div>
            </div>

            <div class="row">
              <div class="col-12">
                <p v-if="item.email_headers.length === 0 && !item.showHeaders">
                  <a href="#" @click.prevent="() => showSMTPHeaders(n)">
                    <oat-icon icon="plus" />{{ $t('settings.smtp.setCustomHeaders') }}</a>
                </p>
                <oat-field v-if="item.email_headers.length > 0 || item.showHeaders"
                  :message="$t('settings.smtp.customHeadersHelp')">
                  <textarea aria-label="field" v-model="item.strEmailHeaders" name="email_headers"
                    placeholder="[{&quot;X-Custom&quot;: &quot;value&quot;}, {&quot;X-Custom2&quot;: &quot;value&quot;}]" />
                </oat-field>
              </div>
            </div>
            <hr />

            <form @submit.prevent="() => doSMTPTest(item, n)">
              <div class="row">
                <template v-if="smtpTestItem === n">
                  <div class="col-5">
                    <strong>{{ $t('settings.general.fromEmail') }}</strong>
                    <br />
                    {{ settings['app.from_email'] }}
                  </div>
                  <div class="col-4">
                    <oat-field :label="$t('settings.smtp.toEmail')">
                      <input aria-label="field" type="email" required v-model="testEmail" :ref="'testEmailTo'"
                        placeholder="email@site.com">
                    </oat-field>
                  </div>
                </template>
                <div class="col-3 align-right">
                  <button type="button" v-if="smtpTestItem === n" data-variant="primary" @click.prevent="() => doSMTPTest(item, n)">
                    {{ $t('settings.smtp.sendTest') }}
                  </button>
                  <a href="#" v-else data-variant="primary" @click.prevent="showTestForm(n)">
                    <oat-icon icon="rocket-launch-outline" /> {{ $t('settings.smtp.testConnection') }}
                  </a>
                </div>
                <div class="row">
                  <div class="col-12" />
                </div>
              </div>
              <div v-if="errMsg && smtpTestItem === n">
                <oat-field class="mt-4">
                  <textarea aria-label="field" v-model="errMsg" readonly />
                </oat-field>
              </div>
            </form><!-- smtp test -->
          </div>
        </div><!-- second container col-12 -->
      </div><!-- block -->
    </div><!-- mail-servers -->

    <button type="button" @click="addSMTP" data-variant="primary">
      {{ $t('globals.buttons.addNew') }}
    </button>
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
  azure: {
    host: 'smtp.azurecomm.net', port: 587, auth_protocol: 'login', tls_type: 'STARTTLS',
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
  forwardemail: {
    host: 'smtp.forwardemail.net', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  postmark: {
    host: 'smtp.postmarkapp.com', port: 587, auth_protocol: 'cram', tls_type: 'STARTTLS',
  },
  lettermint: {
    host: 'smtp.lettermint.co', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
};

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
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
        name: '',
        enabled: true,
        host: '',
        hello_hostname: '',
        port: 587,
        auth_protocol: 'none',
        username: '',
        password: '',
        email_headers: [],
        from_addresses: [],
        max_conns: 10,
        max_msg_retries: 2,
        msg_retry_delay: '0s',
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
        this.$utils.toast(this.$t('settings.smtp.testEnterEmail'), '');
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

    validateFromAddress(v) {
      // Accept an e-mail address (user@example.com) or a domain (example.com).
      return /^[^\s@]+(\.[^\s@]+)+$|^[^\s@]+@[^\s@]+(\.[^\s@]+)+$/.test(v);
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
