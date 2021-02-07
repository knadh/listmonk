<template>
  <section>
    <form @submit.prevent="onSubmit">
      <div class="modal-card content template-modal-content" style="width: auto">
        <header class="modal-card-head">
            <b-button @click="previewTemplate"
              class="is-pulled-right" type="is-primary"
              icon-left="file-find-outline">{{ $t('templates.preview') }}</b-button>

            <h4 v-if="isEditing">{{ data.name }}</h4>
            <h4 v-else>{{ $t('templates.newTemplate') }}</h4>
        </header>
        <section expanded class="modal-card-body">
            <b-field :label="$t('globals.fields.name')" label-position="on-border">
            <b-input :maxlength="200" :ref="'focus'" v-model="form.name"
                placeholder="$t('globals.fields.name')" required></b-input>
            </b-field>

            <b-field :label="$t('globals.fields.rawHTML')" label-position="on-border">
            <b-input v-model="form.body" type="textarea" required />
            </b-field>

            <p class="is-size-7">
              {{ $t('templates.placeholderHelp', { placeholder: egPlaceholder }) }}
              <a target="_blank" href="https://listmonk.app/docs/templating">
                {{ $t('globals.buttons.learnMore') }}
              </a>
            </p>
        </section>
        <footer class="modal-card-foot has-text-right">
            <b-button @click="$parent.close()">{{ $t('globals.buttons.close') }}</b-button>
            <b-button native-type="submit" type="is-primary"
            :loading="loading.templates">{{ $t('globals.buttons.save') }}</b-button>
        </footer>
      </div>
    </form>
    <campaign-preview v-if="previewItem"
      type='template'
      :title="previewItem.name"
      :body="form.body"
      @close="closePreview"></campaign-preview>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CampaignPreview from '../components/CampaignPreview.vue';

export default Vue.extend({
  components: {
    CampaignPreview,
  },

  props: {
    data: Object,
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
      previewItem: null,
      egPlaceholder: '{{ template "content" . }}',
    };
  },

  methods: {
    previewTemplate() {
      this.previewItem = this.data;
    },

    closePreview() {
      this.previewItem = null;
    },

    onSubmit() {
      if (this.isEditing) {
        this.updateTemplate();
        return;
      }

      this.createTemplate();
    },

    createTemplate() {
      const data = {
        id: this.data.id,
        name: this.form.name,
        body: this.form.body,
      };

      this.$api.createTemplate(data).then((d) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: this.$t('globals.messages.created', { name: d.name }),
          type: 'is-success',
          queue: false,
        });
      });
    },

    updateTemplate() {
      const data = {
        id: this.data.id,
        name: this.form.name,
        body: this.form.body,
      };

      this.$api.updateTemplate(data).then((d) => {
        this.$emit('finished');
        this.$parent.close();
        this.$buefy.toast.open({
          message: `'${d.name}' updated`,
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
