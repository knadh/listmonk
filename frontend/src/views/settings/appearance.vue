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
        <div class="block">
          <p>
            {{ $t('settings.appearance.templates.help') }}
          </p>
          <p>
            <a href="https://github.com/knadh/listmonk/blob/master/static/email-templates/base.html">https://github.com/knadh/listmonk/blob/master/static/email-templates/base.html</a>
          </p>
          <p>
            {{ $t('settings.appearance.templates.moreHelp') }}
          </p>

          <div class="columns">
            <div class="column is-three-quarters">
              <p>
                {{ $t('settings.appearance.templates.previewHelp') }}
              </p>
            </div>
            <div class="column has-text-right">
              <!-- eslint-disable-next-line max-len -->
              <b-button type="is-primary" class="has-text-right" icon-left="file-find-outline" @click.prevent="showPreview()">{{ $t('globals.buttons.preview') }}</b-button>
            </div>
          </div>
        </div>
        <hr>

        <div class="block">
          <!-- eslint-disable-next-line max-len -->
          <b-field :label="$t('settings.appearance.headerTemplate')" label-position="on-border" :message="$t('settings.appearance.headerTemplateHelp')">
            <!-- eslint-disable-next-line max-len -->
            <b-input v-model="data['appearance.admin.templates.header']" type="textarea" name="body" />
          </b-field>

          <div class="columns">
            <div class="column is-three-quarters">
              <p>
                {{ $t('settings.appearance.headerTemplateInfo') }}
              </p>
            </div>
            <div class="column has-text-right">
              <!-- eslint-disable-next-line max-len -->
              <b-button type="is-primary" size="is-small" class="has-text-right" icon-left="eye-outline" @click.prevent="showDefaultTemplate('header')">{{ $t('globals.buttons.showDefault') }}</b-button>
            </div>
          </div>
        </div>
        <hr>

        <div class="block">
          <!-- eslint-disable-next-line max-len -->
          <b-field :label="$t('settings.appearance.footerTemplate')" label-position="on-border" :message="$t('settings.appearance.footerTemplateHelp')">
            <!-- eslint-disable-next-line max-len -->
            <b-input v-model="data['appearance.admin.templates.footer']" type="textarea" name="body" />
          </b-field>

          <div class="columns">
            <div class="column is-three-quarters">
              <p>
                {{ $t('settings.appearance.footerTemplateInfo') }}
              </p>
            </div>
            <div class="column has-text-right">
              <!-- eslint-disable-next-line max-len -->
              <b-button type="is-primary" size="is-small" class="has-text-right" icon-left="eye-outline" @click.prevent="showDefaultTemplate('footer')">{{ $t('globals.buttons.showDefault') }}</b-button>
            </div>
          </div>
        </div>

        <!-- show default modal -->
        <!-- eslint-disable-next-line max-len -->
        <b-modal scroll="keep" :aria-modal="true" :active.sync="isDefaultViewerVisible" :maxwidth="1200">
          <appearance-default-viewer :defaultName="defaultName" :defaultBody="defaultBody" />
        </b-modal>

        <!-- show preview -->
        <!-- eslint-disable-next-line max-len -->
        <appearance-notif-preview v-if="previewTitle" :previewTitle="previewTitle" @close="closePreview"></appearance-notif-preview>
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
  },

  data() {
    return {
      data: this.form,
      activeTab: 0,
      isDefaultViewerVisible: false,
      defaultName: '',
      defaultBody: '',
      previewTitle: null,
    };
  },

  watch: {
    activeTab: function activeTab() {
      localStorage.setItem('admin.settings.appearance.active_tab', this.activeTab);
    },
  },

  methods: {
    showPreview() {
      this.previewTitle = this.$t('settings.appearance.templates.previewTitle');
    },

    closePreview() {
      this.previewTitle = null;
    },

    showDefaultTemplate(name) {
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
  },

  computed: {
    ...mapState(['settings']),
  },
});

</script>
