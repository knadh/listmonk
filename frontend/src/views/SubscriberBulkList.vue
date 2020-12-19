<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card" style="width: auto">
      <header class="modal-card-head">
        <h4 class="title is-size-5">{{ $t('subscribers.manageLists') }}</h4>
      </header>

      <section expanded class="modal-card-body">
        <b-field label="Action">
          <div>
            <b-radio v-model="form.action" name="action" native-value="add">
              {{ $t('globals.buttons.add') }}
            </b-radio>
            <b-radio v-model="form.action" name="action" native-value="remove">
              {{ $t('globals.buttons.remove') }}
            </b-radio>
            <b-radio
              v-model="form.action"
              name="action"
              native-value="unsubscribe"
            >{{ $t('subscribers.markUnsubscribed') }}</b-radio>
          </div>
        </b-field>

        <list-selector
          label="Target lists"
          placeholder="Lists to apply to"
          v-model="form.lists"
          :selected="form.lists"
          :all="lists.results"
        ></list-selector>
      </section>

      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">{{ $t('globals.buttons.close') }}</b-button>
        <b-button native-type="submit" type="is-primary"
          :disabled="form.lists.length === 0">{{ $t('globals.buttons.save') }}</b-button>
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
    numSubscribers: Number,
  },

  data() {
    return {
      // Binds form input values.
      form: {
        action: 'add',
        lists: [],
      },
    };
  },

  methods: {
    onSubmit() {
      this.$emit('finished', this.form.action, this.form.lists);
      this.$parent.close();
    },
  },

  computed: {
    ...mapState(['lists', 'loading']),
  },
});
</script>
