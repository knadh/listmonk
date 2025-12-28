<template>
  <div>
    <div class="items webhooks">
      <div class="block box" v-for="(item, n) in data.webhooks" :key="n">
        <div class="columns">
          <div class="column is-2">
            <b-field :label="$t('globals.buttons.enabled')">
              <b-switch v-model="item.enabled" name="enabled" :native-value="true" />
            </b-field>
            <b-field>
              <a @click.prevent="$utils.confirm(null, () => removeWebhook(n))" href="#" class="is-size-7">
                <b-icon icon="trash-can-outline" size="is-small" />
                {{ $t('globals.buttons.delete') }}
              </a>
            </b-field>
          </div><!-- first column -->

          <div class="column" :class="{ disabled: !item.enabled }">
            <div class="columns">
              <div class="column is-4">
                <b-field :label="$t('globals.fields.name')" label-position="on-border"
                  :message="$t('settings.webhooks.nameHelp')">
                  <b-input v-model="item.name" name="name" placeholder="my-webhook" :maxlength="200" />
                </b-field>
              </div>
              <div class="column is-8">
                <b-field :label="$t('settings.webhooks.url')" label-position="on-border"
                  :message="$t('settings.webhooks.urlHelp')">
                  <b-input v-model="item.url" name="url" placeholder="https://example.com/webhook"
                    :maxlength="2000" expanded type="url" pattern="https?://.*" />
                </b-field>
              </div>
            </div><!-- name and url -->

            <div class="columns">
              <div class="column">
                <b-field :label="$t('settings.webhooks.events')" label-position="on-border"
                  :message="$t('settings.webhooks.eventsHelp')">
                  <b-taginput
                    v-model="item.events"
                    :data="filteredEvents"
                    autocomplete
                    :allow-new="false"
                    :open-on-focus="true"
                    field="name"
                    icon="label"
                    @typing="filterEvents"
                  >
                    <template #default="props">
                      <span>{{ props.option }}</span>
                    </template>
                    <template #empty>
                      {{ $t('globals.messages.noResults') }}
                    </template>
                  </b-taginput>
                </b-field>
              </div>
            </div><!-- events -->

            <hr />

            <div class="columns">
              <div class="column is-4">
                <b-field :label="$t('settings.webhooks.authType')" label-position="on-border"
                  :message="$t('settings.webhooks.authTypeHelp')">
                  <b-select v-model="item.auth_type" name="auth_type" expanded>
                    <option value="none">{{ $t('settings.webhooks.authNone') }}</option>
                    <option value="basic">{{ $t('settings.webhooks.authBasic') }}</option>
                    <option value="hmac">{{ $t('settings.webhooks.authHmac') }}</option>
                  </b-select>
                </b-field>
              </div>
              <div class="column is-4" v-if="item.auth_type === 'basic'">
                <b-field :label="$t('settings.webhooks.username')" label-position="on-border">
                  <b-input v-model="item.auth_basic_user" name="auth_basic_user" :maxlength="200" />
                </b-field>
              </div>
              <div class="column is-4" v-if="item.auth_type === 'basic'">
                <b-field :label="$t('settings.webhooks.password')" label-position="on-border"
                  :message="$t('globals.messages.passwordChange')">
                  <b-input v-model="item.auth_basic_pass" name="auth_basic_pass" type="password"
                    :placeholder="$t('globals.messages.passwordChange')" :maxlength="200" />
                </b-field>
              </div>
              <div class="column is-8" v-if="item.auth_type === 'hmac'">
                <b-field :label="$t('settings.webhooks.hmacSecret')" label-position="on-border"
                  :message="$t('settings.webhooks.hmacSecretHelp')">
                  <b-input v-model="item.auth_hmac_secret" name="auth_hmac_secret" type="password"
                    :placeholder="$t('globals.messages.passwordChange')" :maxlength="500" />
                </b-field>
              </div>
            </div><!-- auth -->
            <hr />

            <div class="columns">
              <div class="column is-4">
                <b-field :label="$t('settings.webhooks.maxRetries')" label-position="on-border"
                  :message="$t('settings.webhooks.maxRetriesHelp')">
                  <b-numberinput v-model="item.max_retries" name="max_retries" type="is-light"
                    controls-position="compact" placeholder="3" min="0" max="10" />
                </b-field>
              </div>
              <div class="column is-4">
                <b-field :label="$t('settings.webhooks.timeout')" label-position="on-border"
                  :message="$t('settings.webhooks.timeoutHelp')">
                  <b-input v-model="item.timeout" name="timeout" placeholder="30s"
                    :pattern="regDuration" :maxlength="10" />
                </b-field>
              </div>
            </div>
          </div>
        </div><!-- second container column -->
      </div><!-- block -->
    </div><!-- webhooks -->

    <b-button @click="addWebhook" icon-left="plus" type="is-primary">
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
      type: Object, default: () => { },
    },
    events: {
      type: Array, default: () => [],
    },
  },

  data() {
    return {
      data: this.form,
      regDuration,
      filteredEvents: this.events,
    };
  },

  methods: {
    addWebhook() {
      this.data.webhooks.push({
        enabled: true,
        name: '',
        url: '',
        events: [],
        auth_type: 'none',
        auth_basic_user: '',
        auth_basic_pass: '',
        auth_hmac_secret: '',
        max_retries: 3,
        timeout: '30s',
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.webhooks input[name="name"]');
        items[items.length - 1].focus();
      });
    },

    removeWebhook(i) {
      this.data.webhooks.splice(i, 1);
    },

    filterEvents(text) {
      this.filteredEvents = this.events.filter((option) => option.toLowerCase().indexOf(text.toLowerCase()) >= 0);
    },
  },
});
</script>
