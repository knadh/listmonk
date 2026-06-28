<template>
  <form @submit.prevent="onSubmit">
    <div class="dialog-card" style="width: auto">
      <header class="dialog-head">
        <h4>
          {{ $t('subscribers.manageLists') }}
        </h4>
      </header>

      <section class="dialog-body">
        <b-field label="Action">
          <div>
            <b-radio v-model="form.action" name="action" native-value="add" data-cy="check-list-add">
              {{ $t('globals.buttons.add') }}
            </b-radio>
            <b-radio v-model="form.action" name="action" native-value="remove" data-cy="check-list-remove">
              {{ $t('globals.buttons.remove') }}
            </b-radio>
            <b-radio v-model="form.action" name="action" native-value="unsubscribe" data-cy="check-list-unsubscribe">
              {{ $t('subscribers.markUnsubscribed') }}
            </b-radio>
          </div>
        </b-field>

        <list-selector label="Target lists" placeholder="Lists to apply to" v-model="form.lists" :selected="form.lists"
          :all="lists.results" />

        <b-field :message="$t('subscribers.preconfirmHelp')">
          <b-checkbox v-model="form.preconfirm" data-cy="preconfirm" :native-value="true" :disabled="!hasOptinList">
            {{ $t('subscribers.preconfirm') }}
          </b-checkbox>
        </b-field>
      </section>

      <footer class="dialog-foot align-right">
        <button type="button" class="outline" @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </button>
        <button type="submit" data-variant="primary" :disabled="form.lists.length === 0">
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

export default Vue.extend({
  components: {
    ListSelector,
  },

  props: {
    numSubscribers: { type: Number, default: 0 },
  },

  data() {
    return {
      // Binds form input values.
      form: {
        action: 'add',
        lists: [],
        preconfirm: false,
      },
    };
  },

  methods: {
    onSubmit() {
      this.$emit('finished', this.form.action, this.form.preconfirm, this.form.lists);
      this.$parent.close();
    },
  },

  computed: {
    ...mapState(['lists', 'loading']),

    hasOptinList() {
      return this.form.lists.some((l) => l.optin === 'double');
    },
  },
});
</script>
