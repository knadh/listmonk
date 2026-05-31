<template>
  <div>
    <div class="row mb-6">
      <div class="col-4">
        <oat-field data-cy="btn-enable-bounce">
          <oat-switch v-model="data['bounce.enabled']" name="bounce.enabled">
            {{ $t('settings.bounces.enable') }}
          </oat-switch>
        </oat-field>
      </div>
      <div class="col-8">
        <div v-for="typ in bounceTypes" :key="typ" class="row">
          <div class="col-2" :class="{ disabled: !data['bounce.enabled'] }" :label="$t('settings.bounces.count')">
            {{ $t(`bounces.${typ}`) }}
          </div>
          <div class="col-4" :class="{ disabled: !data['bounce.enabled'] }">
            <oat-field :label="$t('settings.bounces.count')" :message="$t('settings.bounces.countHelp')"
              data-cy="btn-bounce-count">
              <input aria-label="field" type="number" v-model.number="data['bounce.actions'][typ]['count']"
                name="bounce.count" placeholder="3" min="1" max="1000">
            </oat-field>
          </div>
          <div class="col-4" :class="{ disabled: !data['bounce.enabled'] }">
            <oat-field :label="$t('settings.bounces.action')">
              <select aria-label="field" name="bounce.action" v-model="data['bounce.actions'][typ]['action']">
                <option value="none">
                  {{ $t('globals.terms.none') }}
                </option>
                <option value="unsubscribe">
                  {{ $t('email.unsub') }}
                </option>
                <option value="blocklist">
                  {{ $t('settings.bounces.blocklist') }}
                </option>
                <option value="delete">
                  {{ $t('globals.buttons.delete') }}
                </option>
              </select>
            </oat-field>
          </div>
        </div>
      </div>
    </div><!-- row -->

    <div class="mb-6">
      <oat-field data-cy="btn-enable-bounce-webhook">
        <oat-switch v-model="data['bounce.webhooks_enabled']" :disabled="!data['bounce.enabled']"
          name="webhooks_enabled" :native-value="true" data-cy="btn-enable-bounce-webhook">
          {{ $t('settings.bounces.enableWebhooks') }}
        </oat-switch>
        <p class="text-light text-7">
          <a href="https://listmonk.app/docs/bounces" target="_blank" rel="noopener noreferer">{{
            $t('globals.buttons.learnMore') }} &rarr;</a>
        </p>
      </oat-field>
      <div class="card" v-if="data['bounce.webhooks_enabled']">
        <div class="row">
          <div class="col-12">
            <oat-field>
              <oat-switch v-model="data['bounce.ses_enabled']" name="ses_enabled" :native-value="true"
                data-cy="btn-enable-bounce-ses">
                {{ $t('settings.bounces.enableSES') }}
              </oat-switch>
            </oat-field>
          </div>
        </div>
        <div class="row">
          <div class="col-3">
            <oat-field>
              <oat-switch v-model="data['bounce.sendgrid_enabled']" name="sendgrid_enabled" :native-value="true"
                data-cy="btn-enable-bounce-sendgrid">
                {{ $t('settings.bounces.enableSendgrid') }}
              </oat-switch>
            </oat-field>
          </div>
          <div class="col-9">
            <oat-field :label="$t('settings.bounces.sendgridKey')" :message="$t('globals.messages.passwordChange')">
              <input aria-label="field" v-model="data['bounce.sendgrid_key']" type="password"
                :disabled="!data['bounce.sendgrid_enabled']" name="sendgrid_enabled" :native-value="true"
                data-cy="btn-enable-bounce-sendgrid">
            </oat-field>
          </div>
        </div>
        <div class="row">
          <div class="col-3">
            <oat-field>
              <oat-switch v-model="data['bounce.postmark'].enabled" name="postmark_enabled" :native-value="true"
                data-cy="btn-enable-bounce-postmark">
                {{ $t('settings.bounces.enablePostmark') }}
              </oat-switch>
            </oat-field>
          </div>
          <div class="col-4">
            <oat-field :label="$t('settings.bounces.postmarkUsername')"
              :message="$t('settings.bounces.postmarkUsernameHelp')">
              <input aria-label="field" v-model="data['bounce.postmark'].username" type="text"
                :disabled="!data['bounce.postmark'].enabled" name="postmark_username"
                data-cy="btn-enable-bounce-postmark">
            </oat-field>
          </div>
          <div class="col-5">
            <oat-field :label="$t('settings.bounces.postmarkPassword')"
              :message="$t('globals.messages.passwordChange')">
              <input aria-label="field" v-model="data['bounce.postmark'].password" type="password"
                :disabled="!data['bounce.postmark'].enabled" name="postmark_password"
                data-cy="btn-enable-bounce-postmark">
            </oat-field>
          </div>
        </div>
        <div class="row">
          <div class="col-3">
            <oat-field>
              <oat-switch v-model="data['bounce.forwardemail'].enabled" name="forwardemail_enabled" :native-value="true"
                data-cy="btn-enable-bounce-forwardemail">
                {{ $t('settings.bounces.enableForwardemail') }}
              </oat-switch>
            </oat-field>
          </div>
          <div class="col-9">
            <oat-field :label="$t('settings.bounces.forwardemailKey')" :message="$t('globals.messages.passwordChange')">
              <input aria-label="field" v-model="data['bounce.forwardemail'].key" type="password"
                :disabled="!data['bounce.forwardemail'].enabled" name="forwardemail_enabled" :native-value="true"
                data-cy="btn-enable-bounce-forwardemail">
            </oat-field>
          </div>
        </div>
        <div class="row">
          <div class="col-3">
            <oat-field>
              <oat-switch v-model="data['bounce.lettermint'].enabled" name="lettermint_enabled" :native-value="true"
                data-cy="btn-enable-bounce-lettermint">
                {{ $t('settings.bounces.enableLettermint') }}
              </oat-switch>
            </oat-field>
          </div>
          <div class="col-9">
            <oat-field :label="$t('settings.bounces.lettermintKey')" :message="$t('globals.messages.passwordChange')">
              <input aria-label="field" v-model="data['bounce.lettermint'].key" type="password"
                :disabled="!data['bounce.lettermint'].enabled" name="lettermint_key" data-cy="bounce-lettermint-key">
            </oat-field>
          </div>
        </div>
      </div>
    </div>

    <!-- bounce mailcard -->
    <oat-field>
      <oat-switch v-if="data['bounce.mailcardes']" v-model="data['bounce.mailcardes'][0].enabled"
        :disabled="!data['bounce.enabled']" name="enabled" :native-value="true" data-cy="btn-enable-bounce-mailcard">
        {{ $t('settings.bounces.enableMailcard') }}
      </oat-switch>
    </oat-field>

    <template v-if="data['bounce.enabled'] && data['bounce.mailcardes'][0].enabled">
      <div class="card" v-for="(item, n) in data['bounce.mailcardes']" :key="n">
        <div class="row">
          <div class="col-12" :class="{ disabled: !item.enabled }">
            <div class="row">
              <div class="col-3">
                <oat-field :label="$t('settings.bounces.type')">
                  <select aria-label="field" v-model="item.type" name="type">
                    <option value="pop">
                      POP
                    </option>
                  </select>
                </oat-field>
              </div>
              <div class="col-6">
                <oat-field :label="$t('settings.mailserver.host')" :message="$t('settings.mailserver.hostHelp')">
                  <input aria-label="field" v-model="item.host" name="host" placeholder="bounce.yourmailserver.net"
                    :maxlength="200">
                </oat-field>
              </div>
              <div class="col-3">
                <oat-field :label="$t('settings.mailserver.port')" :message="$t('settings.mailserver.portHelp')">
                  <input aria-label="field" type="number" v-model.number="item.port" name="port" placeholder="25"
                    min="1" max="65535">
                </oat-field>
              </div>
            </div><!-- host -->

            <div class="row">
              <div class="col-3">
                <oat-field :label="$t('settings.mailserver.authProtocol')">
                  <select aria-label="field" v-model="item.auth_protocol" name="auth_protocol">
                    <option value="none">
                      none
                    </option>
                    <option v-if="item.type === 'pop'" value="userpass">
                      userpass
                    </option>
                    <template v-else>
                      <option value="cram">
                        cram
                      </option>
                      <option value="plain">
                        plain
                      </option>
                      <option value="login">
                        login
                      </option>
                    </template>
                  </select>
                </oat-field>
              </div>
              <div class="col-9">
                <oat-field>
                  <oat-field :label="$t('settings.mailserver.username')">
                    <input aria-label="field" v-model="item.username" :disabled="item.auth_protocol === 'none'"
                      name="username" placeholder="mysmtp" :maxlength="200">
                  </oat-field>
                  <oat-field :label="$t('settings.mailserver.password')"
                    :message="$t('settings.mailserver.passwordHelp')">
                    <input aria-label="field" v-model="item.password" :disabled="item.auth_protocol === 'none'"
                      name="password" type="password" :placeholder="$t('settings.mailserver.passwordHelp')"
                      :maxlength="200">
                  </oat-field>
                </oat-field>
              </div>
            </div><!-- auth -->

            <div class="row">
              <div class="col-6">
                <oat-field>
                  <oat-field :message="$t('settings.mailserver.tlsHelp')">
                    <oat-switch v-model="item.tls_enabled" name="item.tls_enabled">
                      {{ $t('settings.mailserver.tls') }}
                    </oat-switch>
                  </oat-field>
                  <oat-field :message="$t('settings.mailserver.skipTLSHelp')">
                    <oat-switch v-model="item.tls_skip_verify" :disabled="!item.tls_enabled"
                      name="item.tls_skip_verify">
                      {{ $t('settings.mailserver.skipTLS') }}
                    </oat-switch>
                  </oat-field>
                </oat-field>
              </div>
              <div class="col-2" />
              <div class="col-4">
                <oat-field :label="$t('settings.bounces.scanInterval')"
                  :message="$t('settings.bounces.scanIntervalHelp')">
                  <input aria-label="field" v-model="item.scan_interval" name="scan_interval" placeholder="15m"
                    :pattern="regDuration" :maxlength="10">
                </oat-field>
              </div>
            </div><!-- TLS -->
          </div>
        </div><!-- second container col-12 -->
      </div><!-- block -->
    </template>
  </div>
</template>

<script>
import Vue from 'vue';
import { regDuration } from '../../constants';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      bounceTypes: ['soft', 'hard', 'complaint'],
      data: this.form,
      regDuration,
    };
  },

  methods: {
    removeBounceBox(i) {
      this.data['bounce.mailcardes'].splice(i, 1);
    },
  },
});
</script>
