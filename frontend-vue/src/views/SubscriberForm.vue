<template>
  <form @submit.prevent="onSubmit">
    <div class="dialog-card content" style="width: auto">
      <header class="dialog-head">
        <b-tag v-if="isEditing" :type="data.status" class="float-right">
          {{ $t(`subscribers.status.${data.status}`) }}
        </b-tag>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ $t('subscribers.newSubscriber') }}
        </h4>

        <p v-if="isEditing" class="text-light text-7 ">
          {{ $t('globals.fields.id') }}: <span data-cy="id"><copy-text :text="`${data.id}`" /></span>
          {{ $t('globals.fields.uuid') }}: <copy-text :text="data.uuid" />
        </p>
      </header>

      <section class="dialog-body">
        <b-field :label="$t('subscribers.email')">
          <input aria-label="field" :maxlength="200" v-model="form.email" name="email" :ref="'focus'"
            :placeholder="$t('subscribers.email')" required>
        </b-field>

        <div class="row">
          <div class="col-8">
            <b-field :label="$t('globals.fields.name')">
              <input aria-label="field" :maxlength="200" v-model="form.name" name="name"
                :placeholder="$t('globals.fields.name')">
            </b-field>
          </div>
          <div class="col-4">
            <b-field :label="$t('globals.fields.status')" :message="$t('subscribers.blocklistedHelp')">
              <select aria-label="field" v-model="form.status" name="status" :placeholder="$t('globals.fields.status')"
                required>
                <option value="enabled">
                  {{ $t('subscribers.status.enabled') }}
                </option>
                <option value="blocklisted">
                  {{ $t('subscribers.status.blocklisted') }}
                </option>
              </select>
            </b-field>
          </div>
        </div>

        <ot-tabs>
          <div role="tablist">
            <button type="button" role="tab">{{ $t('globals.terms.lists') }}</button>
            <button type="button" role="tab" :disabled="!data.lists || data.lists.length === 0">
              {{ `${$tc('globals.terms.subscriptions', 2)} (${data.lists ? data.lists.length : 0})` }}
            </button>
            <button type="button" role="tab" :disabled="bounces.length === 0">
              {{ `${$t('globals.terms.bounces')} (${bounces.length})` }}
            </button>
            <button type="button" role="tab" :disabled="!isEditing">{{ $t('subscribers.activity') }}</button>
          </div>

          <section role="tabpanel">
            <list-selector :label="$t('subscribers.lists')" :placeholder="$t('subscribers.listsPlaceholder')"
              :message="$t('subscribers.listsHelp')" v-model="form.lists" :selected="form.lists" :all="lists.results" />
            <div class="row">
              <div class="col-7">
                <b-field :message="$t('subscribers.preconfirmHelp')">
                  <b-checkbox v-model="form.preconfirm" :native-value="true" :disabled="!hasOptinList">
                    {{ $t('subscribers.preconfirm') }}
                  </b-checkbox>
                </b-field>
              </div>
              <div v-if="$can('subscribers:manage') && isEditing" class="col-5 align-right">
                <a href="#" @click.prevent="sendOptinConfirmation" :class="{ 'disabled': !hasOptinList }">
                  <b-icon icon="email-outline" />
                  {{ $t('subscribers.sendOptinConfirm') }}</a>
              </div>
            </div>
          </section><!-- lists -->

          <section role="tabpanel">
            <template v-if="data.lists">
              <b-table :data="data.lists" default-sort="createdAt" class="subscriptions">
                <b-table-column v-slot="props" field="name" :label="$tc('globals.terms.list', 1)">
                  <div>
                    <a v-if="!props.row.restricted" :href="`/admin/lists/${props.row.id}`">
                      {{ props.row.name }}
                    </a>
                    <span v-else class="text-lighter text-7 ">{{ props.row.name }}</span>
                    <br />
                    <b-tag :type="props.row.optin" :data-cy="`optin-${props.row.optin}`">
                      <b-icon
                        :icon="props.row.optin === 'double' ? 'account-check-outline' : 'account-off-outline'" />
                      {{ ' ' }}
                      {{ $t(`lists.optins.${props.row.optin}`) }}
                    </b-tag>{{ ' ' }}
                  </div>
                </b-table-column>

                <b-table-column v-slot="props" field="status" cell-class="status"
                  :label="$t('globals.fields.status')">
                  <b-tag :type="`status-${props.row.subscriptionStatus}`">
                    {{ $t(`subscribers.status.${props.row.subscriptionStatus}`) }}
                  </b-tag>
                  <template v-if="props.row.optin === 'double' && props.row.subscriptionMeta.optinIp">
                    <br /><span class="text-7">{{ props.row.subscriptionMeta.optinIp }}</span>
                  </template>
                </b-table-column>

                <b-table-column v-slot="props" field="createdAt" :label="$t('globals.fields.createdAt')">
                  {{ $utils.niceDate(props.row.subscriptionCreatedAt, true) }}
                </b-table-column>

                <b-table-column v-slot="props" field="updatedAt" :label="$t('globals.fields.updatedAt')">
                  {{ $utils.niceDate(props.row.subscriptionCreatedAt, true) }}
                </b-table-column>
              </b-table>
            </template>
          </section><!-- subscriptions -->

          <section role="tabpanel" class="bounces">
            <a href="#" class="text-6 align-right" disabed="true" @click.prevent="deleteBounces" v-if="isBounceVisible">
              <b-icon icon="trash-can-outline" />
              {{ $t('globals.buttons.delete') }}
            </a>

            <b-table :data="bounces" default-sort="createdAt" class="bounces">
              <b-table-column field="campaign" :label="$tc('globals.terms.campaign', 1)" v-slot="props">
                <div v-if="props.row.campaign">
                  <router-link :to="{ name: 'bounces', query: { campaign_id: props.row.campaign.id } }">
                    {{ props.row.campaign.name }}
                  </router-link>
                </div>
              </b-table-column>

              <b-table-column field="createdAt" :label="$t('globals.fields.createdAt')" v-slot="props">
                {{ $utils.niceDate(props.row.createdAt, true) }}
              </b-table-column>

              <b-table-column field="action" :label="$t('globals.fields.type')" v-slot="props">
                <span class="align-right">
                  <a href="#" @click.prevent="toggleMeta(props.row.id)">
                    {{ props.row.source }}
                    <b-icon :icon="visibleMeta[props.row.id] ? 'arrow-up' : 'arrow-down'" />
                  </a>
                </span>
                <span class="clearfix" />
                <pre v-if="visibleMeta[props.row.id]">{{ props.row.meta }}</pre>
              </b-table-column>
            </b-table>
          </section><!-- bounces -->

          <section role="tabpanel" class="activity">
            <subscriber-activity v-if="isEditing && data.id" :subscriber-id="data.id" />
          </section><!-- activity -->
        </ot-tabs>

        <b-field :message="$t('subscribers.attribsHelp') + ' ' + egAttribs" class="mt-6">
          <div>
            <h5>{{ $t('globals.terms.attribs') }}</h5>
            <textarea aria-label="field" v-model="form.strAttribs" name="attribs" />
            <a href="https://listmonk.app/docs/concepts" target="_blank" rel="noopener noreferrer" class="text-7">
              {{ $t('globals.buttons.learnMore') }} <b-icon icon="link-variant" />
            </a>
          </div>
        </b-field>
      </section>
      <footer class="dialog-foot align-right">
        <button type="button" class="outline" @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </button>
        <button v-if="$can('subscribers:manage')" type="submit" data-variant="primary" :loading="loading.subscribers">
          {{ $t('globals.buttons.save') }}
        </button>
      </footer>
    </div>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListSelector from '../components/ListSelector.vue';
import CopyText from '../components/CopyText.vue';
import SubscriberActivity from '../components/SubscriberActivity.vue';

export default Vue.extend({
  components: {
    ListSelector,
    CopyText,
    SubscriberActivity,
  },

  props: {
    data: {
      type: Object,
      default: () => ({ lists: [] }),
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
        this.$utils.toast(
          `${this.$t('subscribers.invalidJSON')}: ${e.toString()}`,
          '',

          3000,
        );
        return null;
      }
      if (attribs instanceof Array) {
        this.$utils.toast('Attributes should be a map {} and not an array []', '', 3000);
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
