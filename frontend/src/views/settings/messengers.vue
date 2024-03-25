<template>
  <div>
    <div class="items messengers">
      <div class="block box" v-for="(item, n) in data.messengers" :key="n">
        <div class="columns">
          <div class="column is-2">
            <b-field :label="$t('globals.buttons.enabled')">
              <b-switch v-model="item.enabled" name="enabled" :native-value="true" />
            </b-field>
            <b-field>
              <a @click.prevent="$utils.confirm(null, () => removeMessenger(n))" href="#" class="is-size-7">
                <b-icon icon="trash-can-outline" size="is-small" />
                {{ $t('globals.buttons.delete') }}
              </a>
            </b-field>
          </div><!-- first column -->

          <div class="column" :class="{ disabled: !item.enabled }">
            <div class="columns">
              <div class="column is-4">
                <b-field :label="$t('globals.fields.name')" label-position="on-border"
                  :message="$t('settings.messengers.nameHelp')">
                  <b-input v-model="item.name" name="name" placeholder="mymessenger" :maxlength="200" />
                </b-field>
              </div>
              <div class="column is-8">
                <b-field :label="$t('settings.messengers.url')" label-position="on-border"
                  :message="$t('settings.messengers.urlHelp')">
                  <b-input v-model="item.root_url" name="root_url" placeholder="https://postback.messenger.net/path"
                    :maxlength="200" expanded type="url" pattern="https?://.*" />
                </b-field>
              </div>
            </div><!-- host -->

            <div class="columns">
              <div class="column">
                <b-field grouped>
                  <b-field :label="$t('settings.messengers.username')" label-position="on-border" expanded>
                    <b-input v-model="item.username" name="username" :maxlength="200" />
                  </b-field>
                  <b-field :label="$t('settings.messengers.password')" label-position="on-border" expanded
                    :message="$t('globals.messages.passwordChange')">
                    <b-input v-model="item.password" name="password" type="password"
                      :placeholder="$t('globals.messages.passwordChange')" :maxlength="200" />
                  </b-field>
                </b-field>
              </div>
            </div><!-- auth -->
            <hr />

            <div class="columns">
              <div class="column is-4">
                <b-field :label="$t('settings.messengers.maxConns')" label-position="on-border"
                  :message="$t('settings.messengers.maxConnsHelp')">
                  <b-numberinput v-model="item.max_conns" name="max_conns" type="is-light" controls-position="compact"
                    placeholder="25" min="1" max="65535" />
                </b-field>
              </div>
              <div class="column is-4">
                <b-field :label="$t('settings.messengers.retries')" label-position="on-border"
                  :message="$t('settings.messengers.retriesHelp')">
                  <b-numberinput v-model="item.max_msg_retries" name="max_msg_retries" type="is-light"
                    controls-position="compact" placeholder="2" min="1" max="1000" />
                </b-field>
              </div>
              <div class="column is-4">
                <b-field :label="$t('settings.messengers.timeout')" label-position="on-border"
                  :message="$t('settings.messengers.timeoutHelp')">
                  <b-input v-model="item.timeout" name="timeout" placeholder="5s" :pattern="regDuration"
                    :maxlength="10" />
                </b-field>
              </div>
            </div>
            <hr />
          </div>
        </div><!-- second container column -->
      </div><!-- block -->
    </div><!-- mail-servers -->

    <b-button @click="addMessenger" icon-left="plus" type="is-primary">
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
  },

  data() {
    return {
      data: this.form,
      regDuration,
    };
  },

  methods: {
    addMessenger() {
      this.data.messengers.push({
        enabled: true,
        root_url: '',
        name: '',
        username: '',
        password: '',
        max_conns: 25,
        max_msg_retries: 2,
        timeout: '5s',
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.messengers input[name="name"]');
        items[items.length - 1].focus();
      });
    },

    removeMessenger(i) {
      this.data.messengers.splice(i, 1);
    },
  },
});
</script>
