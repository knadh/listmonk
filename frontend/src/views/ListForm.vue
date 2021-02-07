<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">
        <p v-if="isEditing" class="has-text-grey-light is-size-7">
          {{ $t('globals.fields.id') }}: {{ data.id }} /
          {{ $t('globals.fields.uuid') }}: {{ data.uuid }}
        </p>
        <b-tag v-if="isEditing" :class="[data.type, 'is-pulled-right']">{{ data.type }}</b-tag>
        <h4 v-if="isEditing">{{ data.name }}</h4>
        <h4 v-else>{{ $t('lists.newList') }}</h4>
      </header>
      <section expanded class="modal-card-body">
        <b-field :label="$t('globals.fields.name')" label-position="on-border">
          <b-input :maxlength="200" :ref="'focus'" v-model="form.name"
            :placeholder="$t('globals.fields.name')" required></b-input>
        </b-field>

        <b-field :label="$t('lists.type')" label-position="on-border"
          :message="$t('lists.typeHelp')">
          <b-select v-model="form.type" :placeholder="$t('lists.typeHelp')" required>
            <option value="private">{{ $t('lists.types.private') }}</option>
            <option value="public">{{ $t('lists.types.public') }}</option>
          </b-select>
        </b-field>

        <b-field :label="$t('lists.optin')" label-position="on-border"
          :message="$t('lists.optinHelp')">
          <b-select v-model="form.optin" placeholder="Opt-in type" required>
            <option value="single">{{ $t('lists.optins.single') }}</option>
            <option value="double">{{ $t('lists.optins.double') }}</option>
          </b-select>
        </b-field>

        <b-field :label="$t('globals.terms.tags')" label-position="on-border">
          <b-taginput v-model="form.tags" ellipsis
            icon="tag-outline" :placeholder="$t('globals.terms.tags')"></b-taginput>
        </b-field>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">{{ $t('globals.buttons.close') }}</b-button>
        <b-button native-type="submit" type="is-primary"
          :loading="loading.lists">{{ $t('globals.buttons.save') }}</b-button>
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
        type: 'private',
        optin: 'single',
        tags: [],
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
      this.$api.createList(this.form).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: this.$t('globals.messages.created', { name: data.name }),
          type: 'is-success',
          queue: false,
        });
      });
    },

    updateList() {
      this.$api.updateList({ id: this.data.id, ...this.form }).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: this.$t('globals.messages.updated', { name: data.name }),
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
    this.form = { ...this.form, ...this.$props.data };

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
