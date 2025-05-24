<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">
        <p v-if="isEditing" class="has-text-grey-light is-size-7">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
        </p>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ $t('users.newUser') }}
        </h4>
      </header>
      <section expanded class="modal-card-body">
        <div class="columns">
          <div class="column is-6">
            <b-field label-position="on-border" class="mb-6">
              <b-radio-button v-model="form.type" name="type" native-value="user" :disabled="isEditing" expanded
                type="is-light is-outlined">
                <b-icon icon="account-outline" />
                {{ $t('users.type.user') }}
              </b-radio-button>
              <b-radio-button v-model="form.type" name="type" native-value="api" :disabled="isEditing" expanded
                type="is-light is-outlined">
                <b-icon icon="code" />
                {{ $t('users.type.api') }}
              </b-radio-button>
            </b-field>
          </div>
          <div class="column is-6">
            <b-field :label="$t('globals.fields.status')" label-position="on-border">
              <b-select v-model="form.status" name="status" required expanded>
                <option value="enabled">
                  {{ $t('users.status.enabled') }}
                </option>
                <option value="disabled">
                  {{ $t('users.status.disabled') }}
                </option>
              </b-select>
            </b-field>
          </div>
        </div>

        <b-field :label="$t('users.username')" label-position="on-border">
          <b-input :maxlength="200" v-model="form.username" name="username" ref="focus" autofocus
            :placeholder="$t('users.username')" required :message="$t('users.usernameHelp')" autocomplete="off"
            pattern="[a-zA-Z0-9_\-\.@]+$" />
        </b-field>

        <b-field :label="$t('globals.fields.name')" label-position="on-border">
          <b-input :maxlength="200" v-model="form.name" name="name" :placeholder="$t('globals.fields.name')" />
        </b-field>

        <b-field v-if="form.type !== 'api'" :label="$t('subscribers.email')" label-position="on-border">
          <b-input :maxlength="200" v-model="form.email" name="email" :placeholder="$t('subscribers.email')" required />
        </b-field>

        <template v-if="form.type !== 'api'">
          <div class="box">
            <b-field>
              <b-checkbox v-model="form.passwordLogin" :native-value="true" name="password_login">
                {{ $t('users.passwordEnable') }}
              </b-checkbox>
            </b-field>

            <div class="columns">
              <div class="column is-6">
                <b-field :label="$t('users.password')" label-position="on-border">
                  <b-input :disabled="!form.passwordLogin" minlength="8" :maxlength="200" v-model="form.password"
                    type="password" name="password" :placeholder="$t('users.password')"
                    :required="form.passwordLogin && !isEditing" />
                </b-field>
              </div>
              <div class="column is-6">
                <b-field :label="$t('users.passwordRepeat')" label-position="on-border">
                  <b-input :disabled="!form.passwordLogin" minlength="8" :maxlength="200" v-model="form.password2"
                    type="password" name="password2" :required="form.passwordLogin && !isEditing && form.password" />
                </b-field>
              </div>
            </div>
          </div>
        </template>

        <h5>{{ $tc('users.roles') }}</h5>
        <div class="box">
          <div class="columns">
            <div class="column is-6">
              <b-field :label="$tc('users.userRole')" label-position="on-border">
                <b-select v-model="form.userRoleId" name="user_role" required expanded>
                  <option v-for="r in userRoles" :value="r.id" :key="r.id">
                    {{ r.name }}
                  </option>
                </b-select>
              </b-field>
            </div>

            <div class="column is-6">
              <b-field :label="$tc('users.listRole', 0)" label-position="on-border">
                <b-select v-model="form.listRoleId" name="list_role" expanded>
                  <option value="">&mdash; {{ $t("globals.terms.none") }} &mdash;</option>
                  <option v-for="r in listRoles" :value="r.id" :key="r.id">
                    {{ r.name }}
                  </option>
                </b-select>
              </b-field>
            </div>
          </div>
        </div>

        <div v-if="apiToken" class="user-api-token">
          <p>{{ $t('users.apiOneTimeToken') }}</p>
          <copy-text :text="apiToken" />
        </div>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </b-button>
        <b-button v-if="$can('users:manage') && !apiToken" native-type="submit" type="is-primary"
          :loading="loading.lists" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </footer>
    </div>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CopyText from '../components/CopyText.vue';

export default Vue.extend({
  name: 'UserForm',

  components: {
    CopyText,
  },

  props: {
    data: { type: Object, default: () => ({}) },
    isEditing: { type: Boolean, default: false },
  },

  data() {
    return {
      // Binds form input values.
      form: {
        username: '',
        email: '',
        name: '',
        password: '',
        passwordLogin: false,
        type: 'user',
        status: 'enabled',
      },
      apiToken: null,
    };
  },

  methods: {
    onSubmit() {
      if (!this.form.passwordLogin) {
        this.form.password = null;
        this.form.password2 = null;
      }

      if (this.isEditing) {
        if (this.form.type !== 'api' && this.form.passwordLogin && this.form.password && this.form.password !== this.form.password2) {
          this.$utils.toast(this.$t('users.passwordMismatch'), 'is-danger');
          return;
        }

        this.updateUser();
        return;
      }

      if (this.form.type !== 'api' && this.form.passwordLogin && this.form.password !== this.form.password2) {
        this.$utils.toast(this.$t('users.passwordMismatch'), 'is-danger');
        return;
      }

      this.createUser();
    },

    createUser() {
      const form = {
        ...this.form, password_login: this.form.passwordLogin, user_role_id: this.form.userRoleId, list_role_id: this.form.listRoleId || null,
      };
      this.$api.createUser(form).then((data) => {
        this.$emit('finished');
        this.$utils.toast(this.$t('globals.messages.created', { name: data.name }));

        // If the user is an API user, show the one-time token.
        if (form.type === 'api') {
          this.apiToken = data.password;
          return;
        }

        this.$emit('finished');
        this.$parent.close();
      });
    },

    updateUser() {
      const form = {
        ...this.form, password_login: this.form.passwordLogin, user_role_id: this.form.userRoleId, list_role_id: this.form.listRoleId || null,
      };
      this.$api.updateUser({ id: this.data.id, ...form }).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.updated', { name: data.name }));
      });
    },

    hasType(t) {
      // If the user being edited is API, then the only valid field is API.
      // Otherwise, all fields are valid except API.
      return !this.$props.isEditing || (this.form.type === 'api' ? t === 'api' : t !== 'api');
    },
  },

  computed: {
    ...mapState(['loading', 'userRoles', 'listRoles']),
  },

  mounted() {
    this.form = { ...this.form, ...this.$props.data };
    if (this.$props.data.userRole) {
      this.form.userRoleId = this.$props.data.userRole.id;
    }

    this.form.listRoleId = this.$props.data.listRole ? this.$props.data.listRole.id : '';

    this.$api.getUserRoles();
    this.$api.getListRoles();

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
