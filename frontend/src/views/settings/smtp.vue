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
                href="#" class="is-size-7" data-cy="btn-delete-smtp">
                <b-icon icon="trash-can-outline" size="is-small" />
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
                    <b-input v-model="item.username"
                      :disabled="item.auth_protocol === 'none'"
                      name="username" placeholder="mysmtp" :maxlength="200" />
                  </b-field>
                  <b-field :label="$t('settings.mailserver.password')"
                    label-position="on-border" expanded
                    :message="$t('settings.mailserver.passwordHelp')">
                    <b-input v-model="item.password"
                      :disabled="item.auth_protocol === 'none'"
                      name="password" type="password"
                      :placeholder="$t('settings.mailserver.passwordHelp')"
                      :maxlength="200" />
                  </b-field>
                </b-field>
              </div>
            </div><!-- auth -->
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
            <hr />

            <div>
              <p v-if="item.email_headers.length === 0 && !item.showHeaders">
                <a href="#" class="is-size-7" @click.prevent="() => showSMTPHeaders(n)">
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
import { regDuration } from '../../constants';

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
  },
});
</script>
