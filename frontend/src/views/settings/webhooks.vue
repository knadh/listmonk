<template>
  <div>
    <div class="columns">
      <div class="column is-6">
        <p class="has-text-grey">
          {{ $t('settings.webhooks.help') }}
        </p>
      </div>
    </div>

    <hr />

    <!-- Subscription Confirmed Webhook -->
    <div class="columns mb-6">
      <div class="column is-12">
        <h4 class="title is-5">{{ $t('settings.webhooks.subscriptionConfirmed') }}</h4>
        <p class="has-text-grey mb-4">{{ $t('settings.webhooks.subscriptionConfirmedHelp') }}</p>

        <b-field :label="$t('globals.buttons.enable')">
          <b-switch v-model="data['webhooks'].subscription_confirmed.enabled"
            name="webhooks_subscription_enabled" />
        </b-field>

        <div class="box" v-if="data['webhooks'].subscription_confirmed.enabled">
          <div class="columns">
            <div class="column is-8">
              <b-field :label="$t('globals.terms.url')" label-position="on-border">
                <b-input v-model="data['webhooks'].subscription_confirmed.url"
                  name="webhooks_subscription_url"
                  placeholder="https://your-server.com/webhook/subscription"
                  type="url" />
              </b-field>
            </div>
          </div>

          <div class="columns">
            <div class="column is-3">
              <b-field :label="$t('settings.webhooks.authType')" label-position="on-border">
                <b-select v-model="data['webhooks'].subscription_confirmed.auth_type"
                  name="webhooks_auth_type" expanded>
                  <option value="none">{{ $t('globals.terms.none') }}</option>
                  <option value="basic">Basic Auth</option>
                  <option value="bearer">Bearer Token</option>
                </b-select>
              </b-field>
            </div>

            <template v-if="data['webhooks'].subscription_confirmed.auth_type === 'basic'">
              <div class="column is-4">
                <b-field :label="$t('settings.mailserver.username')" label-position="on-border">
                  <b-input v-model="data['webhooks'].subscription_confirmed.username"
                    name="webhooks_username" />
                </b-field>
              </div>
              <div class="column is-4">
                <b-field :label="$t('settings.mailserver.password')" label-position="on-border"
                  :message="$t('globals.messages.passwordChange')">
                  <b-input v-model="data['webhooks'].subscription_confirmed.password"
                    name="webhooks_password" type="password" />
                </b-field>
              </div>
            </template>

            <template v-if="data['webhooks'].subscription_confirmed.auth_type === 'bearer'">
              <div class="column is-8">
                <b-field :label="$t('settings.webhooks.bearerToken')" label-position="on-border"
                  :message="$t('globals.messages.passwordChange')">
                  <b-input v-model="data['webhooks'].subscription_confirmed.bearer_token"
                    name="webhooks_bearer_token" type="password" />
                </b-field>
              </div>
            </template>
          </div>

          <div class="columns">
            <div class="column is-3">
              <b-field :label="$t('settings.webhooks.timeout')" label-position="on-border"
                :message="$t('settings.webhooks.timeoutHelp')">
                <b-input v-model="data['webhooks'].subscription_confirmed.timeout"
                  name="webhooks_timeout" placeholder="10s" :pattern="regDuration" />
              </b-field>
            </div>
            <div class="column is-3">
              <b-field :label="$t('settings.webhooks.maxRetries')" label-position="on-border"
                :message="$t('settings.webhooks.maxRetriesHelp')">
                <b-numberinput v-model="data['webhooks'].subscription_confirmed.max_retries"
                  name="webhooks_max_retries" type="is-light" controls-position="compact"
                  min="1" max="10" />
              </b-field>
            </div>
          </div>
        </div>
      </div>
    </div>
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
      data: this.form,
      regDuration,
    };
  },
});
</script>
