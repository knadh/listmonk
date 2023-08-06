<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">
        <b-tag v-if="isEditing" :class="[data.status, 'is-pulled-right']">
          {{ $t(`subscribers.status.${data.status}`) }}
        </b-tag>
        <h4 v-if="isEditing">{{ data.name }}</h4>
        <h4 v-else>{{ $t('subscribers.newSubscriber') }}</h4>

        <p v-if="isEditing" class="has-text-grey is-size-7">
          {{ $t('globals.fields.id') }}: <span data-cy="id">{{ data.id }}</span> /
          {{ $t('globals.fields.uuid') }}: {{ data.uuid }}
        </p>
      </header>

      <section expanded class="modal-card-body">
        <b-field :label="$t('subscribers.email')" label-position="on-border">
          <b-input :maxlength="200" v-model="form.email" name="email" :ref="'focus'"
            :placeholder="$t('subscribers.email')" required></b-input>
        </b-field>

        <div class="columns">
          <div class="column is-8">
            <b-field :label="$t('globals.fields.name')" label-position="on-border">
              <b-input :maxlength="200" v-model="form.name" name="name"
                :placeholder="$t('globals.fields.name')"></b-input>
            </b-field>
          </div>
          <div class="column is-4">
            <b-field :label="$t('globals.fields.status')" label-position="on-border"
              :message="$t('subscribers.blocklistedHelp')">
              <b-select v-model="form.status" name="status"
                :placeholder="$t('globals.fields.status')" required expanded>
                <option value="enabled">{{ $t('subscribers.status.enabled') }}</option>
                <option value="blocklisted">{{ $t('subscribers.status.blocklisted') }}</option>
              </b-select>
            </b-field>
          </div>
        </div>

        <list-selector
          :label="$t('subscribers.lists')"
          :placeholder="$t('subscribers.listsPlaceholder')"
          :message="$t('subscribers.listsHelp')"
          v-model="form.lists"
          :selected="form.lists"
          :all="lists.results"
        ></list-selector>
        <div class="columns mb-5">
          <div class="column is-7">
            <b-field :message="$t('subscribers.preconfirmHelp')">
                <b-checkbox v-model="form.preconfirm"
                  :native-value="true" :disabled="!hasOptinList">
                  {{ $t('subscribers.preconfirm') }}
                </b-checkbox>
            </b-field>
          </div>
          <div class="column is-5 has-text-right" v-if="isEditing">
            <a href="" @click.prevent="sendOptinConfirmation"
              :class="{'is-disabled': !hasOptinList}">
              <b-icon icon="email-outline" size="is-small" />
              {{ $t('subscribers.sendOptinConfirm') }}</a>
          </div>
        </div>

        <b-field :message="$t('subscribers.attribsHelp') + ' ' + egAttribs" class="mb-5">
          <div>
            <h5>{{ $t('subscribers.attribs') }}</h5>
            <b-input v-model="form.strAttribs" name="attribs" type="textarea" />
            <a href="https://listmonk.app/docs/concepts"
              target="_blank" rel="noopener noreferrer" class="is-size-7">
              {{ $t('globals.buttons.learnMore') }} <b-icon icon="link-variant" size="is-small" />
            </a>
          </div>
        </b-field>

        <div class="mb-5" v-if="data.lists">
          <h5>{{ $tc('globals.terms.subscriptions', 2) }} ({{ data.lists.length }})</h5>
          <b-table :data="data.lists" hoverable default-sort="createdAt" class="subscriptions"
          >
            <b-table-column v-slot="props" field="name"
              :label="$tc('globals.terms.list', 1)">
              <div>
                <router-link :to="`/lists/${props.row.id}`">
                  {{ props.row.name }}
                </router-link>
                <br />
                <b-tag :class="props.row.optin" :data-cy="`optin-${props.row.optin}`">
                  <b-icon :icon="props.row.optin === 'double' ?
                    'account-check-outline' : 'account-off-outline'" size="is-small" />
                  {{ ' ' }}
                  {{ $t(`lists.optins.${props.row.optin}`) }}
                </b-tag>{{ ' ' }}
              </div>
            </b-table-column>

            <b-table-column v-slot="props" field="status" cell-class="status"
              :label="$t('globals.fields.status')">
              <b-tag :class="`status-${props.row.subscriptionStatus}`">
                {{ $t(`subscribers.status.${props.row.subscriptionStatus}`) }}
              </b-tag>
              <template v-if="props.row.optin === 'double'
                && props.row.subscriptionMeta.optinIp">
                <br /><span class="is-size-7">{{ props.row.subscriptionMeta.optinIp }}</span>
              </template>
            </b-table-column>

            <b-table-column v-slot="props" field="createdAt"
              :label="$t('globals.fields.createdAt')">
              {{ $utils.niceDate(props.row.subscriptionCreatedAt, true) }}
            </b-table-column>

            <b-table-column v-slot="props" field="updatedAt"
              :label="$t('globals.fields.updatedAt')">
              {{ $utils.niceDate(props.row.subscriptionCreatedAt, true) }}
            </b-table-column>
          </b-table>
        </div>

        <div class="bounces" v-show="bounces.length > 0">
          <a href="#" class="is-size-6" disabed="true"
            @click.prevent="toggleBounces">
            <b-icon icon="email-bounce"></b-icon>
            {{ $t('bounces.view') }} ({{ bounces.length }})
          </a>
          <a href="#" class="is-size-6 is-pulled-right" disabed="true"
            @click.prevent="deleteBounces" v-if="isBounceVisible">
            <b-icon icon="trash-can-outline"></b-icon>
            {{ $t('globals.buttons.delete') }}
          </a>

          <div v-if="isBounceVisible" class="mt-4">
            <ol class="is-size-7">
              <li v-for="b in bounces" :key="b.id" class="mb-2">
                  <div v-if="b.campaign">
                    <router-link :to="{ name: 'bounces', query: { campaign_id: b.campaign.id } }">
                      {{ b.campaign.name }}
                    </router-link>
                  </div>
                  {{ $utils.niceDate(b.createdAt, true) }}
                  <span class="is-pulled-right">
                    <a href="#" @click.prevent="toggleMeta(b.id)">
                      {{ b.source }}
                      <b-icon :icon="visibleMeta[b.id] ? 'arrow-up' : 'arrow-down'" />
                    </a>
                  </span>
                  <span class="is-clearfix"></span>
                  <pre v-if="visibleMeta[b.id]">{{ b.meta }}</pre>
              </li>
            </ol>
          </div>
        </div>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">{{ $t('globals.buttons.close') }}</b-button>
        <b-button native-type="submit" type="is-primary"
          :loading="loading.subscribers">{{ $t('globals.buttons.save') }}</b-button>
      </footer>
    </div>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListSelector from '../components/ListSelector.vue';

export default Vue.extend({
  components: {
    ListSelector,
  },

  props: {
    data: {
      type: Object,
      default: () => {},
    },
    isEditing: Boolean,
  },

  data() {
    return {
      // Binds form input values. This is populated by subscriber props passed
      // from the parent component in mounted().
      form: {
        lists: [],
        strAttribs: '{}',
        status: 'enabled',
        preconfirm: false,
      },
      isBounceVisible: false,
      bounces: [],
      visibleMeta: {},

      egAttribs: '{"job": "developer", "location": "Mars", "has_rocket": true}',
    };
  },

  methods: {
    toggleBounces() {
      this.isBounceVisible = !this.isBounceVisible;
    },

    toggleMeta(id) {
      let v = false;
      if (!this.visibleMeta[id]) {
        v = true;
      }
      Vue.set(this.visibleMeta, id, v);
    },

    deleteBounces(sub) {
      this.$utils.confirm(
        null,
        () => {
          this.$api.deleteSubscriberBounces(this.form.id).then(() => {
            this.getBounces();
            this.$utils.toast(this.$t('globals.messages.deleted', { name: sub.name }));
          });
        },
      );
    },

    getBounces() {
      this.$api.getSubscriberBounces(this.form.id).then((data) => {
        this.bounces = data;
      });
    },

    onSubmit() {
      // If there is no name, auto-generate one from the e-mail.
      if (!this.form.name) {
        let name = '';
        [name] = this.form.email.toLowerCase().split('@');

        if (name.includes('.')) {
          this.form.name = name.split('.').map((c) => this.$utils.titleCase(c)).join(' ');
        } else {
          this.form.name = this.$utils.titleCase(name);
        }
      }

      if (this.isEditing) {
        this.updateSubscriber();
        return;
      }

      this.createSubscriber();
    },

    createSubscriber() {
      let attribs = {};
      if (this.form.strAttribs) {
        attribs = this.validateAttribs(this.form.strAttribs);
        if (!attribs) {
          return;
        }
      }

      const data = {
        email: this.form.email,
        name: this.form.name,
        status: this.form.status,
        attribs,
        preconfirm_subscriptions: this.form.preconfirm,

        // List IDs.
        lists: this.form.lists.map((l) => l.id),
      };

      this.$api.createSubscriber(data).then((d) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.created', { name: d.name }));
      });
    },

    updateSubscriber() {
      let attribs = {};
      if (this.form.strAttribs) {
        attribs = this.validateAttribs(this.form.strAttribs);
        if (!attribs) {
          return;
        }
      }

      const data = {
        id: this.form.id,
        email: this.form.email,
        name: this.form.name,
        status: this.form.status,
        preconfirm_subscriptions: this.form.preconfirm,
        attribs,

        // List IDs.
        lists: this.form.lists.map((l) => l.id),
      };

      this.$api.updateSubscriber(data).then((d) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.updated', { name: d.name }));
      });
    },

    sendOptinConfirmation() {
      this.$api.sendSubscriberOptin(this.form.id).then(() => {
        this.$utils.toast(this.$t('subscribers.sentOptinConfirm'));
      });
    },

    validateAttribs(str) {
      // Parse and validate attributes JSON.
      let attribs = {};
      try {
        attribs = JSON.parse(str);
      } catch (e) {
        this.$utils.toast(`${this.$t('subscribers.invalidJSON')}: ${e.toString()}`,
          'is-danger', 3000);
        return null;
      }
      if (attribs instanceof Array) {
        this.$utils.toast('Attributes should be a map {} and not an array []', 'is-danger', 3000);
        return null;
      }

      return attribs;
    },
  },

  computed: {
    ...mapState(['lists', 'loading']),

    hasOptinList() {
      return this.form.lists.some((l) => l.optin === 'double');
    },
  },

  mounted() {
    if (this.$props.isEditing) {
      this.form = {
        ...this.$props.data,

        // Deep-copy the lists array on to the form.
        strAttribs: JSON.stringify(this.$props.data.attribs, null, 4),
      };
    }

    if (this.form.id) {
      this.getBounces();
    }

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
