<template>
  <div class="items">
    <b-tabs type="is-boxed" :animated="false" v-model="activeTab">
      <!-- eslint-disable-next-line max-len -->
      <b-tab-item :label="$t('settings.appearance.admin.name')" label-position="on-border">
        <div class="block">
          <p>
            {{ $t('settings.appearance.admin.help') }}
          </p>
        </div>
        <br /><br />

        <!-- eslint-disable-next-line max-len -->
        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border" :message="$t('settings.appearance.cssHelp')">
          <!-- eslint-disable-next-line max-len -->
          <b-input v-model="data['appearance.admin.custom_css']" type="textarea" name="body" />
        </b-field>
      </b-tab-item><!-- admin -->

      <!-- eslint-disable-next-line max-len -->
      <b-tab-item :label="$t('settings.appearance.public.name')" label-position="on-border">
        <div class="block">
          <p>
            {{ $t('settings.appearance.public.help') }}
          </p>
        </div>
        <br /><br />

        <!-- eslint-disable-next-line max-len -->
        <b-field :label="$t('settings.appearance.customCSS')" label-position="on-border" :message="$t('settings.appearance.cssHelp')">
          <!-- eslint-disable-next-line max-len -->
          <b-input v-model="data['appearance.public.custom_css']" type="textarea" name="body" />
        </b-field>

        <!-- eslint-disable-next-line max-len -->
        <b-field :label="$t('settings.appearance.customJS')" label-position="on-border" :message="$t('settings.appearance.jsHelp')">
          <!-- eslint-disable-next-line max-len -->
          <b-input v-model="data['appearance.public.custom_js']" type="textarea" name="body" />
        </b-field>
      </b-tab-item><!-- public -->

      <!-- eslint-disable-next-line max-len -->
      <b-tab-item :label="$t('settings.appearance.templates.name')" label-position="on-border">
        <p>
          {{ $t('settings.appearance.templates.help') }}
        </p>
        <p>
          <a href="https://listmonk.app/docs/templating/">https://listmonk.app/docs/templating/</a>
        </p>
        <p>
          {{ $t('settings.appearance.templates.previewHelp') }}
        </p>

        <div class="columns is-vcentered">
          <div class="column">
            <b-select v-model="activeTemplate">
              <!-- eslint-disable-next-line max-len -->
              <option v-for="template in definedTemplates" :value="template" :key="template">{{template}}</option>
            </b-select>
          </div>
          <div class="column is-narrow">
            <div class="columns">
              <div class="column is-narrow">
                <!-- eslint-disable-next-line max-len -->
                <b-button type="is-primary is-light" size="is-small" icon-left="eye-outline" @click.prevent="showTemplate(activeTemplate)">{{ $t('globals.buttons.showDefault') }}</b-button>
              </div>
              <div class="column is-narrow">
                <!-- eslint-disable-next-line max-len -->
                <b-button type="is-primary" size="is-small" icon-left="file-find-outline" @click.prevent="showPreview()">{{ $t('globals.buttons.preview') }}</b-button>
              </div>
            </div>
          </div>
        </div>
        <!-- eslint-disable-next-line max-len -->
        <b-field :label="$t('settings.appearance.customTemplate')" label-position="on-border" :message="$t('settings.appearance.templates.templateHelp')">
          <!-- eslint-disable-next-line max-len -->
          <b-input v-model="data[`appearance.admin.custom_templates.${activeTemplate}`]" type="textarea" name="body" />
        </b-field>

        <!-- show default modal -->
        <!-- eslint-disable-next-line max-len -->
        <b-modal scroll="keep" :aria-modal="true" :active.sync="isDefaultViewerVisible" :maxwidth="1200">
          <appearance-default-viewer :defaultName="defaultName" :defaultBody="defaultBody" />
        </b-modal>

        <!-- show preview -->
        <!-- eslint-disable-next-line max-len -->
        <appearance-notif-preview v-if="previewTitle" :previewTitle="previewTitle" :previewURL="previewURL" @close="closePreview"></appearance-notif-preview>
      </b-tab-item><!-- notifications -->
    </b-tabs>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import AppearanceDefaultViewer from './appearanceDefaultViewer.vue';
import AppearanceNotifPreview from './appearanceNotifPreview.vue';

export default Vue.extend({
  components: {
    AppearanceDefaultViewer,
    AppearanceNotifPreview,
  },

  props: {
    form: {
      type: Object,
    },
    definedTemplates: {
      type: Array,
    },
  },

  data() {
    return {
      data: this.form,
      activeTab: 0,
      isDefaultViewerVisible: false,
      defaultName: null,
      defaultBody: null,
      previewTitle: null,
      previewURL: null,
      activeTemplate: null,
    };
  },

  watch: {
    activeTab: function activeTab() {
      localStorage.setItem('admin.settings.appearance.active_tab', this.activeTab);
    },

    activeTemplate: function activeTemplate() {
      localStorage.setItem('admin.settings.appearance.active_template', this.activeTemplate);
    },
  },

  methods: {
    showPreview() {
      this.previewTitle = this.activeTemplate;
      this.previewURL = `/api/admin/templates/preview/${this.activeTemplate}`;
    },

    closePreview() {
      this.previewTitle = null;
      this.previewURL = null;
    },

    showTemplate(name) {
      this.$api.getNotifTemplate(name).then((resp) => {
        const capitalized = name.charAt(0).toUpperCase() + name.slice(1);
        this.defaultName = `{{ ${capitalized} }}`;
        if (resp.length > 0) {
          this.defaultBody = resp;
          this.isDefaultViewerVisible = true;
        }
      });
    },
  },

  mounted() {
    // Reload active tab.
    if (localStorage.getItem('admin.settings.appearance.active_tab')) {
      this.activeTab = JSON.parse(localStorage.getItem('admin.settings.appearance.active_tab'));
    }

    // Reload active Template
    if (localStorage.getItem('admin.settings.appearance.active_template')) {
      this.activeTemplate = localStorage.getItem('admin.settings.appearance.active_template');
    }

    if (!this.activeTemplate) {
      [this.activeTemplate] = this.definedTemplates;
    }
  },

  computed: {
    ...mapState(['settings']),
  },
});

</script>
