<template>
  <section class="user-profile section-mini">
    <b-loading v-if="loading.users" :active="loading.users" :is-full-page="false" />

    <h1 class="title">
      @{{ form.username }}
    </h1>

    <b-tag>{{ form.role.name }}</b-tag>

    <br /><br /><br />
    <form @submit.prevent="onSubmit">
      <b-field v-if="form.type !== 'api'" :label="$t('subscribers.email')" label-position="on-border">
        <b-input :maxlength="200" v-model="form.email" name="email" :placeholder="$t('subscribers.email')"
          :disabled="!form.passwordLogin" required autofocus />
      </b-field>

      <b-field :label="$t('globals.fields.name')" label-position="on-border">
        <b-input :maxlength="200" v-model="form.name" name="name" :placeholder="$t('globals.fields.name')" />
      </b-field>

      <div v-if="form.passwordLogin" class="columns">
        <div class="column is-6">
          <b-field :label="$t('users.password')" label-position="on-border">
            <b-input minlength="8" :maxlength="200" v-model="form.password" type="password" name="password"
              :placeholder="$t('users.password')" />
          </b-field>
        </div>
        <div class="column is-6">
          <b-field :label="$t('users.passwordRepeat')" label-position="on-border">
            <b-input minlength="8" :maxlength="200" v-model="form.password2" type="password" name="password2" />
          </b-field>
        </div>
      </div>

      <b-field expanded>
        <b-button type="is-primary" icon-left="content-save-outline" native-type="submit" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </b-field>
    </form>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  name: 'UserProfile',

  data() {
    return {
      form: {},
    };
  },

  methods: {
    onSubmit() {
      const params = {
        name: this.form.name,
        email: this.form.email,
      };

      if (this.form.passwordLogin && this.form.password) {
        if (this.form.password !== this.form.password2) {
          this.$utils.toast(this.$t('users.passwordMismatch'), 'is-danger');
          return;
        }

        params.password = this.form.password;
        params.password2 = this.form.password2;
      }

      this.$api.updateUserProfile(params).then(() => {
        this.form.password = '';
        this.form.password2 = '';
        this.$utils.toast(this.$t('globals.messages.updated', { name: this.form.username }));
      });
    },
  },

  mounted() {
    this.$api.getUserProfile().then((data) => {
      this.form = data;
    });
  },

  computed: {
    ...mapState(['loading']),
  },

});
</script>
