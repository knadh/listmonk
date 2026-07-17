<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">
        <p v-if="isEditing" class="has-text-grey-light is-size-7">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
          {{ $t('globals.fields.uuid') }}: <copy-text :text="data.uuid" />
        </p>
        <b-tag v-if="isEditing" :class="[data.type, 'is-pulled-right']">
          {{ $t(`lists.types.${data.type}`) }}
        </b-tag>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ $t('lists.newList') }}
        </h4>
      </header>
      <section expanded class="modal-card-body">
        <b-tabs v-model="activeTab" :animated="false">
          <!-- List settings -->
          <b-tab-item :label="$tc('globals.terms.list', 1)">
            <b-field :label="$t('globals.fields.name')" label-position="on-border">
              <b-input :maxlength="200" :ref="'focus'" v-model="form.name" name="name"
                :placeholder="$t('globals.fields.name')" required />
            </b-field>

            <b-field :label="$t('lists.type')" label-position="on-border" :message="$t('lists.typeHelp')">
              <b-select v-model="form.type" name="type" :placeholder="$t('lists.typeHelp')" required expanded>
                <option value="private">
                  {{ $t('lists.types.private') }}
                </option>
                <option value="public">
                  {{ $t('lists.types.public') }}
                </option>
              </b-select>
            </b-field>

            <b-field :label="$t('lists.optin')" label-position="on-border" :message="$t('lists.optinHelp')">
              <b-select v-model="form.optin" name="optin" placeholder="Opt-in type" required expanded>
                <option value="single">
                  {{ $t('lists.optins.single') }}
                </option>
                <option value="double">
                  {{ $t('lists.optins.double') }}
                </option>
              </b-select>
            </b-field>

            <b-field :label="$t('globals.terms.tags')" label-position="on-border">
              <b-taginput v-model="form.tags" name="tags" ellipsis icon="tag-outline"
                :placeholder="$t('globals.terms.tags')" />
            </b-field>

            <b-field :label="$t('globals.fields.description')" label-position="on-border">
              <b-input :maxlength="2000" v-model="form.description" name="description" type="textarea"
                :placeholder="$t('globals.fields.description')" />
            </b-field>

            <b-field :message="$t('lists.archivedHelp')" :label="$t('lists.archived')">
              <b-switch v-model="isArchived" name="status" />
            </b-field>
          </b-tab-item>

          <!-- Welcome e-mail -->
          <b-tab-item :label="$t('lists.welcomeEmail')">
            <b-field :message="$t('lists.welcomeEmailHelp')">
              <b-switch v-model="form.welcomeEnabled" name="welcome_enabled" data-cy="btn-welcome-enabled">
                {{ $t('lists.welcomeEmailEnabled') }}
              </b-switch>
            </b-field>

            <template v-if="form.welcomeEnabled">
              <b-field :label="$t('campaigns.subject')" label-position="on-border">
                <b-input :maxlength="1000" v-model="form.welcomeSubject" name="welcome_subject"
                  :placeholder="$t('campaigns.subject')" required />
              </b-field>

              <editor v-model="form.welcome" :id="data.id || 0" :title="form.name" :templates="templates"
                :content-types="contentTypes" hide-preview />
            </template>
          </b-tab-item>
        </b-tabs>
      </section>
      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </b-button>
        <b-button v-if="$can('lists:manage_all') || $canList(data.id, 'list:manage')" native-type="submit"
          type="is-primary" :loading="loading.lists" data-cy="btn-save">
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
import Editor from '../components/Editor.vue';

export default Vue.extend({
  name: 'ListForm',

  components: {
    CopyText,
    Editor,
  },

  props: {
    data: { type: Object, default: () => ({}) },
    isEditing: { type: Boolean, default: false },
  },

  data() {
    return {
      activeTab: 0,

      contentTypes: Object.freeze({
        richtext: 'Rich text',
        html: 'Raw HTML',
        markdown: 'Markdown',
        plain: 'Plain text',
        visual: 'Visual',
      }),

      // Binds form input values.
      form: {
        name: '',
        type: 'private',
        optin: 'single',
        status: 'active',
        tags: [],

        // Welcome e-mail.
        welcomeEnabled: false,
        welcomeSubject: '',
        welcome: {
          contentType: 'richtext',
          body: '',
          bodySource: null,
          templateId: null,
        },
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

    // payload flattens the welcome sub-form into the snake_case fields the backend expects.
    payload() {
      return {
        name: this.form.name,
        type: this.form.type,
        optin: this.form.optin,
        status: this.form.status,
        tags: this.form.tags,
        description: this.form.description,

        welcome_enabled: this.form.welcomeEnabled,
        welcome_subject: this.form.welcomeSubject,
        welcome_content_type: this.form.welcome.contentType,
        welcome_body: this.form.welcome.body,
        welcome_body_source: this.form.welcome.bodySource,
        welcome_template_id: this.form.welcome.templateId,
      };
    },

    createList() {
      this.$api.createList(this.payload()).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.created', { name: data.name }));
      });
    },

    updateList() {
      this.$api.updateList({ id: this.data.id, ...this.payload() }).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.updated', { name: data.name }));
      });
    },
  },

  computed: {
    ...mapState(['loading', 'profile', 'templates']),

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
    // Merge the incoming list data over the defaults, mapping the flat welcome_* fields
    // (returned camelCased by the API) into the nested `welcome` object the editor binds to.
    const d = this.$props.data;
    this.form = {
      ...this.form,
      ...d,
      welcomeEnabled: d.welcomeEnabled || false,
      welcomeSubject: d.welcomeSubject || '',
      welcome: {
        contentType: d.welcomeContentType || 'richtext',
        body: d.welcomeBody || '',
        bodySource: d.welcomeBodySource || null,
        templateId: d.welcomeTemplateId || null,
      },
    };

    // Load templates for the welcome editor's template selector.
    this.$api.getTemplates();

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
