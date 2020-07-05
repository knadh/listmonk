<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">
        <b-tag v-if="isEditing" :class="[data.type, 'is-pulled-right']">{{ data.type }}</b-tag>
        <h4 v-if="isEditing">{{ data.name }}</h4>
        <h4 v-else>New list</h4>

        <p v-if="isEditing" class="has-text-grey is-size-7">
          ID: {{ data.id }} / UUID: {{ data.uuid }}
        </p>
      </header>
      <section expanded class="modal-card-body">
        <b-field label="Name">
          <b-input :maxlength="200" :ref="'focus'" v-model="form.name"
            placeholder="Name" required></b-input>
        </b-field>

        <b-field label="Type"
          message="Public lists are open to the world to subscribe
                   and their names may appear on public pages such as the subscription
                   management page.">
          <b-select v-model="form.type" placeholder="Type" required>
            <option value="private">Private</option>
            <option value="public">Public</option>
          </b-select>
        </b-field>

        <b-field label="Opt-in"
          message="Double opt-in sends an e-mail to the subscriber asking for
                   confirmation. On Double opt-in lists, campaigns are only sent to
                   confirmed subscribers.">
          <b-select v-model="form.optin" placeholder="Opt-in type" required>
            <option value="single">Single</option>
            <option value="double">Double</option>
          </b-select>
        </b-field>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">Close</b-button>
        <b-button native-type="submit" type="is-primary"
          :loading="loading.lists">Save</b-button>
      </footer>
    </div>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  name: 'ListForm',

  props: {
    data: {},
    isEditing: null,
  },

  data() {
    return {
      // Binds form input values.
      form: {
        name: '',
        type: '',
        optin: '',
      },
    };
  },

  methods: {
    onSubmit() {
      if (this.isEditing) {
        this.updateList();
        return;
      }

      this.createList();
    },

    createList() {
      this.$api.createList(this.form).then((resp) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: `'${resp.data.name}' created`,
          type: 'is-success',
          queue: false,
        });
      });
    },

    updateList() {
      this.$api.updateList({ id: this.data.id, ...this.form }).then((resp) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: `'${resp.data.name}' updated`,
          type: 'is-success',
          queue: false,
        });
      });
    },
  },

  computed: {
    ...mapState(['loading']),
  },

  mounted() {
    this.form = { ...this.$props.data };

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
