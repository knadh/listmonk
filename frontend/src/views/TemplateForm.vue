<template>
  <section>
    <form @submit.prevent="onSubmit">
      <div class="modal-card content template-modal-content" style="width: auto">
        <header class="modal-card-head">
            <b-button @click="previewTemplate"
              class="is-pulled-right" type="is-primary"
              icon-left="file-find-outline">Preview</b-button>

            <h4 v-if="isEditing">{{ data.name }}</h4>
            <h4 v-else>New template</h4>
        </header>
        <section expanded class="modal-card-body">
            <b-field label="Name" label-position="on-border">
            <b-input :maxlength="200" :ref="'focus'" v-model="form.name"
                placeholder="Name" required></b-input>
            </b-field>

            <b-field label="Raw HTML" label-position="on-border">
            <b-input v-model="form.body" type="textarea" required />
            </b-field>

            <p class="is-size-7">
                The placeholder <code>{{ egPlaceholder }}</code>
                should appear in the template.
                <a target="_blank" href="https://listmonk.app/docs/templating">Learn more.</a>
            </p>
        </section>
        <footer class="modal-card-foot has-text-right">
            <b-button @click="$parent.close()">Close</b-button>
            <b-button native-type="submit" type="is-primary"
            :loading="loading.templates">Save</b-button>
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
          message: `'${d.name}' created`,
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
