<template>
  <div class="items">
    <ot-tabs ref="appearanceTabs" class="settings-subtabs" @ot-tab-change="tab = $event.detail.index">
      <div role="tablist">
        <button type="button" role="tab" :aria-selected="tab === 0 ? 'true' : 'false'">
          {{ $t('settings.appearance.adminName') }}
        </button>
        <button type="button" role="tab" :aria-selected="tab === 1 ? 'true' : 'false'">
          {{ $t('settings.appearance.publicName') }}
        </button>
      </div>

      <div role="tabpanel">
        <div>
          {{ $t('settings.appearance.adminHelp') }}
        </div>

        <b-field :label="$t('settings.appearance.customCSS')">
          <code-editor lang="css" v-model="data['appearance.admin.custom_css']" name="body" key="editor-admin-css" />
        </b-field>

        <b-field :label="$t('settings.appearance.customJS')">
          <code-editor lang="javascript" v-model="data['appearance.admin.custom_js']" name="body"
            key="editor-admin-js" />
        </b-field>
      </div>

      <div role="tabpanel">
        <div>
          {{ $t('settings.appearance.publicHelp') }}
        </div>

        <b-field :label="$t('settings.appearance.customCSS')">
          <code-editor lang="css" v-model="data['appearance.public.custom_css']" name="body" key="editor-public-css" />
        </b-field>

        <b-field :label="$t('settings.appearance.customJS')">
          <code-editor lang="javascript" v-model="data['appearance.public.custom_js']" name="body"
            key="editor-public-js" />
        </b-field>
      </div>
    </ot-tabs>
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
    this.$nextTick(() => {
      if (this.$refs.appearanceTabs) {
        this.$refs.appearanceTabs.activeIndex = Number(this.tab);
      }
    });
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
