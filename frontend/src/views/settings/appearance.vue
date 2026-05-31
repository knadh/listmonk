<template>
  <div class="items">
    <div class="settings-subtabs">
      <div role="tablist">
        <button type="button" role="tab" :aria-selected="tab === 0 ? 'true' : 'false'"
          :class="{ outline: tab !== 0 }" @click="tab = 0">
          {{ $t('settings.appearance.adminName') }}
        </button>
        <button type="button" role="tab" :aria-selected="tab === 1 ? 'true' : 'false'"
          :class="{ outline: tab !== 1 }" @click="tab = 1">
          {{ $t('settings.appearance.publicName') }}
        </button>
      </div>

      <div v-show="tab === 0" role="tabpanel">
        <div>
          {{ $t('settings.appearance.adminHelp') }}
        </div>

        <oat-field :label="$t('settings.appearance.customCSS')">
          <code-editor lang="css" v-model="data['appearance.admin.custom_css']" name="body" key="editor-admin-css" />
        </oat-field>

        <oat-field :label="$t('settings.appearance.customJS')">
          <code-editor lang="javascript" v-model="data['appearance.admin.custom_js']" name="body"
            key="editor-admin-js" />
        </oat-field>
      </div>

      <div v-show="tab === 1" role="tabpanel">
        <div>
          {{ $t('settings.appearance.publicHelp') }}
        </div>

        <oat-field :label="$t('settings.appearance.customCSS')">
          <code-editor lang="css" v-model="data['appearance.public.custom_css']" name="body" key="editor-public-css" />
        </oat-field>

        <oat-field :label="$t('settings.appearance.customJS')">
          <code-editor lang="javascript" v-model="data['appearance.public.custom_js']" name="body"
            key="editor-public-js" />
        </oat-field>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CodeEditor from '../../components/CodeEditor.vue';

export default Vue.extend({
  components: {
    'code-editor': CodeEditor,
  },

  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
      tab: 0,
    };
  },

  mounted() {
    this.tab = this.$utils.getPref('settings.apperanceTab') || 0;
  },

  watch: {
    tab(t) {
      this.$utils.setPref('settings.apperanceTab', t);
    },
  },

  computed: {
    ...mapState(['settings']),
  },
});

</script>
