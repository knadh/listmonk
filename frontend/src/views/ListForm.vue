<template>
  <form @submit.prevent="onSubmit">
    <div class="dialog-card content" style="width: auto">
      <header class="dialog-head">
        <p v-if="isEditing" class="text-lighter text-7 ">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
          {{ $t('globals.fields.uuid') }}: <copy-text :text="data.uuid" />
        </p>
        <oat-badge v-if="isEditing" :type="data.type" class="float-right">
          {{ $t(`lists.types.${data.type}`) }}
        </oat-badge>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ $t('lists.newList') }}
        </h4>
      </header>
      <section class="dialog-body">
        <oat-field :label="$t('globals.fields.name')">
          <input aria-label="field" :maxlength="200" :ref="'focus'" v-model="form.name" name="name"
            :placeholder="$t('globals.fields.name')" required>
        </oat-field>

        <oat-field :label="$t('lists.type')" :message="$t('lists.typeHelp')">
          <select aria-label="field" v-model="form.type" name="type" :placeholder="$t('lists.typeHelp')" required>
            <option value="private">
              {{ $t('lists.types.private') }}
            </option>
            <option value="public">
              {{ $t('lists.types.public') }}
            </option>
          </select>
        </oat-field>

        <oat-field :label="$t('lists.optin')" :message="$t('lists.optinHelp')">
          <select aria-label="field" v-model="form.optin" name="optin" placeholder="Opt-in type" required>
            <option value="single">
              {{ $t('lists.optins.single') }}
            </option>
            <option value="double">
              {{ $t('lists.optins.double') }}
            </option>
          </select>
        </oat-field>

        <oat-field :label="$t('globals.terms.tags')">
          <oat-tag-input v-model="form.tags" name="tags" :placeholder="$t('globals.terms.tags')" />
        </oat-field>

        <oat-field :label="$t('globals.fields.description')">
          <textarea aria-label="field" :maxlength="2000" v-model="form.description" name="description"
            :placeholder="$t('globals.fields.description')" />
        </oat-field>

        <oat-field :message="$t('lists.archivedHelp')" :label="$t('lists.archived')">
          <oat-switch v-model="isArchived" name="status" />
        </oat-field>
      </section>
      <footer class="dialog-foot align-right">
        <button type="button" class="outline" @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </button>
        <button v-if="$can('lists:manage_all') || $canList(data.id, 'list:manage')" type="submit" data-variant="primary"
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
  name: 'ListForm',

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
        name: '',
        type: 'private',
        optin: 'single',
        status: 'active',
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
        this.$utils.toast(this.$t('globals.messages.created', { name: data.name }));
      });
    },

    updateList() {
      this.$api.updateList({ id: this.data.id, ...this.form }).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.updated', { name: data.name }));
      });
    },
  },

  computed: {
    ...mapState(['loading', 'profile']),

    isArchived: {
      get() {
        return this.form.status === 'archived';
      },
      set(v) {
        this.form.status = v ? 'archived' : 'active';
      },
    },
  },

  mounted() {
    this.form = { ...this.form, ...this.$props.data };

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
