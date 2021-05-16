<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">

        <b-tag v-if="isEditing" :class="[data.status, 'is-pulled-right']">{{ data.status }}</b-tag>
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

        <b-field :label="$t('globals.fields.name')" label-position="on-border">
          <b-input :maxlength="200" v-model="form.name" name="name"
            :placeholder="$t('globals.fields.name')"></b-input>
        </b-field>

        <b-field :label="$t('globals.fields.status')" label-position="on-border"
          :message="$t('subscribers.blocklistedHelp')">
          <b-select v-model="form.status" name="status" :placeholder="$t('globals.fields.status')"
            required>
            <option value="enabled">{{ $t('subscribers.status.enabled') }}</option>
            <option value="blocklisted">{{ $t('subscribers.status.blocklisted') }}</option>
          </b-select>
        </b-field>

        <list-selector
          :label="$t('subscribers.lists')"
          :placeholder="$t('subscribers.listsPlaceholder')"
          :message="$t('subscribers.listsHelp')"
          v-model="form.lists"
          :selected="form.lists"
          :all="lists.results"
        ></list-selector>

        <b-field :label="$t('subscribers.attribs')" label-position="on-border"
          :message="$t('subscribers.attribsHelp') + ' ' + egAttribs">
          <b-input v-model="form.strAttribs" name="attribs" type="textarea" />
        </b-field>
        <a href="https://listmonk.app/docs/concepts"
          target="_blank" rel="noopener noreferrer" class="is-size-7">
          {{ $t('globals.buttons.learnMore') }} <b-icon icon="link" size="is-small" />.
        </a>
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
      form: { lists: [], strAttribs: '{}' },

      egAttribs: '{"job": "developer", "location": "Mars", "has_rocket": true}',
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
