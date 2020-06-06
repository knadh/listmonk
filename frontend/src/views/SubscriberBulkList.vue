<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card" style="width: auto">
      <header class="modal-card-head">
        <h4>Manage lists</h4>
        <p>{{ numSubscribers }} subscriber(s) selected</p>
      </header>

      <section expanded class="modal-card-body">
        <b-field label="Action">
          <div>
            <b-radio v-model="form.action" name="action" native-value="add">Add</b-radio>
            <b-radio v-model="form.action" name="action" native-value="remove">Remove</b-radio>
            <b-radio
              v-model="form.action"
              name="action"
              native-value="unsubscribe"
            >Mark as unsubscribed</b-radio>
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
        <b-button @click="$parent.close()">Close</b-button>
        <b-button native-type="submit" type="is-primary"
          :disabled="form.lists.length === 0">Save</b-button>
      </footer>
    </div>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListSelector from '../components/ListSelector.vue';

Vue.component('list-selector', ListSelector);

export default Vue.extend({
  name: 'SubscriberBulkList',

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
