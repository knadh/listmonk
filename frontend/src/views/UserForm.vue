<template>
  <form @submit.prevent="onSubmit">
    <div class="dialog-card content" style="width: auto">
      <header class="dialog-head">
        <p v-if="isEditing" class="text-lighter ">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
        </p>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ $t('users.newUser') }}
        </h4>
      </header>
      <section class="dialog-body">
        <div class="row">
          <div class="col-6">
            <oat-field class="mb-6">
              <oat-radio v-model="form.type" name="type" native-value="user" :disabled="isEditing"
>
                <oat-icon icon="account-outline" />
                {{ $t('users.type.user') }}
              </oat-radio>
              <oat-radio v-model="form.type" name="type" native-value="api" :disabled="isEditing"
>
                <oat-icon icon="code" />
                {{ $t('users.type.api') }}
              </oat-radio>
            </oat-field>
          </div>
          <div class="col-6">
            <oat-field :label="$t('globals.fields.status')">
              <select aria-label="field" v-model="form.status" name="status" required>
                <option value="enabled">
                  {{ $t('users.status.enabled') }}
                </option>
                <option value="disabled">
                  {{ $t('users.status.disabled') }}
                </option>
              </select>
            </oat-field>
          </div>
        </div>

        <oat-field :label="$t('users.username')">
          <input aria-label="field" :maxlength="200" v-model="form.username" name="username" ref="focus"
            :placeholder="$t('users.username')" required :message="$t('users.usernameHelp')" autocomplete="off"
            pattern="[a-zA-Z0-9_\-\.@]+$">
        </oat-field>

        <oat-field :label="$t('globals.fields.name')">
          <input aria-label="field" :maxlength="200" v-model="form.name" name="name" :placeholder="$t('globals.fields.name')">
        </oat-field>

        <oat-field v-if="form.type !== 'api'" :label="$t('subscribers.email')">
          <input aria-label="field" :maxlength="200" v-model="form.email" name="email" :placeholder="$t('subscribers.email')" required>
        </oat-field>

        <template v-if="form.type !== 'api'">
          <div class="card">
            <oat-field>
              <oat-checkbox v-model="form.passwordLogin" :native-value="true" name="password_login">
                {{ $t('users.passwordEnable') }}
              </oat-checkbox>
            </oat-field>

            <div class="row">
              <div class="col-6">
                <oat-field :label="$t('users.password')">
                  <input aria-label="field" :disabled="!form.passwordLogin" minlength="8" :maxlength="200" v-model="form.password"
                    type="password" name="password" :placeholder="$t('users.password')"
                    :required="form.passwordLogin && !isEditing">
                </oat-field>
              </div>
              <div class="col-6">
                <oat-field :label="$t('users.passwordRepeat')">
                  <input aria-label="field" :disabled="!form.passwordLogin" minlength="8" :maxlength="200" v-model="form.password2"
                    type="password" name="password2" :required="form.passwordLogin && !isEditing && form.password">
                </oat-field>
              </div>
            </div>
          </div>
        </template>

        <h5>{{ $tc('users.roles') }}</h5>
        <div class="card">
          <div class="row">
            <div class="col-6">
              <oat-field :label="$tc('users.userRole')">
                <select aria-label="field" v-model="form.userRoleId" name="user_role" required>
                  <option v-for="r in userRoles" :value="r.id" :key="r.id">
                    {{ r.name }}
                  </option>
                </select>
              </oat-field>
            </div>

            <div class="col-6">
              <oat-field :label="$tc('users.listRole', 0)">
                <select aria-label="field" v-model="form.listRoleId" name="list_role">
                  <option value="">&mdash; {{ $t("globals.terms.none") }} &mdash;</option>
                  <option v-for="r in listRoles" :value="r.id" :key="r.id">
                    {{ r.name }}
                  </option>
                </select>
              </oat-field>
            </div>
          </div>
        </div>

        <div v-if="apiToken" class="user-api-token">
          <p>{{ $t('users.apiOneTimeToken') }}</p>
          <copy-text :text="apiToken" />
        </div>
      </section>
      <footer class="dialog-foot align-right">
        <button type="button" class="outline" @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </button>
        <button v-if="$can('users:manage') && !apiToken" type="submit" data-variant="primary"
          :loading="loading.lists" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </button>
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
          this.$utils.toast(this.$t('users.passwordMismatch'), '');
          return;
        }

        this.updateUser();
        return;
      }

      if (this.form.type !== 'api' && this.form.passwordLogin && this.form.password !== this.form.password2) {
        this.$utils.toast(this.$t('users.passwordMismatch'), '');
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
