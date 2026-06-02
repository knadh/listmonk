<template>
  <div>
    <div class="items messengers">
      <div class="card" v-for="(item, n) in data.messengers" :key="n">
        <b-field>
          <b-switch v-model="item.enabled" name="enabled" :native-value="true">
            {{ $t('globals.buttons.enabled') }}
          </b-switch>
        </b-field>
        <b-field>
          <a @click.prevent="$utils.confirm(null, () => removeMessenger(n))" href="#">
            <b-icon icon="trash-can-outline" />
            {{ $t('globals.buttons.delete') }}
          </a>
        </b-field>

        <div :class="{ disabled: !item.enabled }">
          <b-field :label="$t('globals.fields.name')" :message="$t('settings.messengers.nameHelp')">
            <input aria-label="field" v-model="item.name" name="name" placeholder="mymessenger" :maxlength="200">
          </b-field>

          <b-field :label="$t('settings.messengers.url')" :message="$t('settings.messengers.urlHelp')">
            <input aria-label="field" v-model="item.root_url" name="root_url"
              placeholder="https://postback.messenger.net/path" :maxlength="200" type="url" pattern="https?://.*">
          </b-field>

          <b-field :label="$t('settings.messengers.username')">
            <input aria-label="field" v-model="item.username" name="username" :maxlength="200">
          </b-field>

          <b-field :label="$t('settings.messengers.password')" :message="$t('globals.messages.passwordChange')">
            <input aria-label="field" v-model="item.password" name="password" type="password"
              :placeholder="$t('globals.messages.passwordChange')" :maxlength="200">
          </b-field>

          <div class="row">
            <div class="col-4">
              <b-field :label="$t('settings.messengers.maxConns')" :message="$t('settings.messengers.maxConnsHelp')">
                <input aria-label="field" type="number" v-model.number="item.max_conns" name="max_conns"
                  placeholder="25" min="1" max="65535">
              </b-field>
            </div>
            <div class="col-4">
              <b-field :label="$t('settings.messengers.retries')" :message="$t('settings.messengers.retriesHelp')">
                <input aria-label="field" type="number" v-model.number="item.max_msg_retries" name="max_msg_retries"
                  placeholder="2" min="1" max="1000">
              </b-field>
            </div>
            <div class="col-4">
              <b-field :label="$t('settings.messengers.timeout')" :message="$t('settings.messengers.timeoutHelp')">
                <input aria-label="field" v-model="item.timeout" name="timeout" placeholder="5s" :pattern="regDuration"
                  :maxlength="10">
              </b-field>
            </div>
          </div>
        </div>
      </div><!-- block -->
    </div><!-- mail-servers -->

    <button type="button" @click="addMessenger" data-variant="primary">
      {{ $t('globals.buttons.addNew') }}
    </button>
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
