<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">

        <b-tag v-if="isEditing" :class="[data.status, 'is-pulled-right']">{{ data.status }}</b-tag>
        <h4 v-if="isEditing">{{ data.name }}</h4>
        <h4 v-else>New subscriber</h4>

        <p v-if="isEditing" class="has-text-grey is-size-7">
          ID: {{ data.id }} / UUID: {{ data.uuid }}
        </p>
      </header>
      <section expanded class="modal-card-body">
        <b-field label="E-mail">
          <b-input :maxlength="200" v-model="form.email" :ref="'focus'"
            placeholder="E-mail" required></b-input>
        </b-field>

        <b-field label="Name">
          <b-input :maxlength="200" v-model="form.name" placeholder="Name"></b-input>
        </b-field>

        <b-field label="Status" message="Blacklisted subscribers will never receive any e-mails.">
          <b-select v-model="form.status" placeholder="Status" required>
            <option value="enabled">Enabled</option>
            <option value="blacklisted">Blacklisted</option>
          </b-select>
        </b-field>

        <list-selector
          label="Lists"
          placeholder="Lists to subscribe to"
          message="Lists from which subscribers have unsubscribed themselves cannot be removed."
          v-model="form.lists"
          :selected="form.lists"
          :all="lists.results"
        ></list-selector>

        <b-field label="Attributes"
          message='Attributes are defined as a JSON map, for example:
            {"job": "developer", "location": "Mars", "has_rocket": true}.'>
          <b-input v-model="form.strAttribs" type="textarea" />
        </b-field>
        <a href="https://listmonk.app/docs/concepts"
          target="_blank" rel="noopener noreferrer" class="is-size-7">
          Learn more <b-icon icon="link" size="is-small" />.
        </a>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">Close</b-button>
        <b-button native-type="submit" type="is-primary"
          :loading="loading.subscribers">Save</b-button>
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
  name: 'SubscriberForm',

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
      form: { lists: [], strAttribs: '{}' },
    };
  },

  methods: {
    onSubmit() {
      if (this.isEditing) {
        this.updateSubscriber();
        return;
      }

      this.createSubscriber();
    },

    createSubscriber() {
      const attribs = this.validateAttribs(this.form.strAttribs);
      if (!attribs) {
        return;
      }

      const data = {
        email: this.form.email,
        name: this.form.name,
        status: this.form.status,
        attribs,

        // List IDs.
        lists: this.form.lists.map((l) => l.id),
      };

      this.$api.createSubscriber(data).then((resp) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: `'${resp.data.name}' created`,
          type: 'is-success',
          queue: false,
        });
      });
    },

    updateSubscriber() {
      const attribs = this.validateAttribs(this.form.strAttribs);
      if (!attribs) {
        return;
      }

      const data = {
        id: this.form.id,
        email: this.form.email,
        name: this.form.name,
        status: this.form.status,
        attribs,

        // List IDs.
        lists: this.form.lists.map((l) => l.id),
      };

      this.$api.updateSubscriber(data).then((resp) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: `'${resp.data.name}' updated`,
          type: 'is-success',
          queue: false,
        });
      });
    },

    validateAttribs(str) {
      // Parse and validate attributes JSON.
      let attribs = {};
      try {
        attribs = JSON.parse(str);
      } catch (e) {
        this.$buefy.toast.open({
          message: `Invalid JSON in attributes: ${e.toString()}`,
          type: 'is-danger',
          duration: 3000,
          queue: false,
        });
        return null;
      }
      if (attribs instanceof Array) {
        this.$buefy.toast.open({
          message: 'Attributes should be a map {} and not an array []',
          type: 'is-danger',
          duration: 3000,
          queue: false,
        });
        return null;
      }

      return attribs;
    },
  },

  computed: {
    ...mapState(['lists', 'loading']),
  },

  mounted() {
    if (this.$props.isEditing) {
      this.form = {
        ...this.$props.data,

        // Deep-copy the lists array on to the form.
        strAttribs: JSON.stringify(this.$props.data.attribs, null, 4),
      };
    }

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
