<template>
  <section class="templates">
    <header class="row page-header">
      <div class="col-10">
        <h1>
          {{ $t('globals.terms.templates') }}
          <span v-if="templates.length > 0">({{ templates.length }})</span>
        </h1>
      </div>
      <div class="col-12 align-right">
        <oat-field v-if="$can('templates:manage')">
          <button type="button" data-variant="primary" class="btn-new" @click="showNewForm">
            {{ $t('globals.buttons.new') }}
          </button>
        </oat-field>
      </div>
    </header>

    <oat-data-table :data="templates" :hoverable="true" :loading="loading.templates" default-sort="createdAt">
      <oat-table-column v-slot="props" field="name" :label="$t('globals.fields.name')" :td-attrs="$utils.tdID" sortable>
        <a href="#" @click.prevent="showEditForm(props.row)">
          {{ props.row.name }}
        </a>
        <span v-if="props.row.isDefault" class="badge secondary">
          {{ $t('templates.default') }}
        </span>

        <p class=" text-light" v-if="props.row.type === 'tx'">
          {{ props.row.subject }}
        </p>
      </oat-table-column>

      <oat-table-column v-slot="props" field="type" :label="$t('globals.fields.type')" sortable>
        <oat-badge v-if="props.row.type === 'campaign'" :type="props.row.type" :data-cy="`type-${props.row.type}`">
          {{ $tc('templates.typeCampaignHTML') }}
        </oat-badge>
        <oat-badge v-else-if="props.row.type === 'campaign_visual'" :type="props.row.type"
          :data-cy="`type-${props.row.type}`">
          {{ $tc('templates.typeCampaignVisual') }}
        </oat-badge>
        <oat-badge v-else :type="props.row.type" :data-cy="`type-${props.row.type}`">
          {{ $tc('templates.typeTransactional') }}
        </oat-badge>
      </oat-table-column>

      <oat-table-column v-slot="props" field="id" :label="$t('globals.fields.id')" sortable>
        {{ props.row.id }}
      </oat-table-column>

      <oat-table-column v-slot="props" field="createdAt" :label="$t('globals.fields.createdAt')" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </oat-table-column>

      <oat-table-column v-slot="props" field="updatedAt" :label="$t('globals.fields.updatedAt')" sortable>
        {{ $utils.niceDate(props.row.updatedAt) }}
      </oat-table-column>

      <oat-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a href="#" @click.prevent="previewTemplate(props.row)" data-cy="btn-preview"
            :aria-label="$t('templates.preview')">

              <oat-icon icon="file-find-outline" />

          </a>
          <a href="#" @click.prevent="showEditForm(props.row)" data-cy="btn-edit"
            :aria-label="$t('globals.buttons.edit')">

              <oat-icon icon="pencil-outline" />

          </a>
          <a href="#" @click.prevent="$utils.prompt(`Clone template`,
            { placeholder: 'Name', value: `Copy of ${props.row.name}` },
            (name) => cloneTemplate(name, props.row))" data-cy="btn-clone" :aria-label="$t('globals.buttons.clone')">

              <oat-icon icon="file-multiple-outline" />

          </a>
          <a v-if="!props.row.isDefault && props.row.type === 'campaign'" href="#"
            @click.prevent="$utils.confirm(null, () => makeTemplateDefault(props.row))" data-cy="btn-set-default"
            :aria-label="$t('templates.makeDefault')">

              <oat-icon icon="check-circle-outline" />

          </a>
          <span v-else class="a text-lighter">
            <oat-icon icon="check-circle-outline" />
          </span>

          <a v-if="!props.row.isDefault" href="#" @click.prevent="$utils.confirm(null, () => deleteTemplate(props.row))"
            data-cy="btn-delete" :aria-label="$t('globals.buttons.delete')">

              <oat-icon icon="trash-can-outline" />

          </a>
          <span v-else class="a text-lighter">
            <oat-icon icon="trash-can-outline" />
          </span>
        </div>
      </oat-table-column>

      <template #empty v-if="!loading.templates">
        <empty-placeholder />
      </template>
</oat-data-table>

    <!-- Add / edit form modal -->
    <oat-modal :active.sync="isFormVisible" :width="1200" :can-cancel="false"
      class="template-modal">
      <template-form :data="curItem" :is-editing="isEditing" @finished="formFinished" />
    </oat-modal>

    <campaign-preview v-if="previewItem" type="template" :id="previewItem.id" :template-type="previewItem.type"
      :title="previewItem.name" @close="closePreview" />
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CampaignPreview from '../components/CampaignPreview.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

import TemplateForm from './TemplateForm.vue';

export default Vue.extend({
  components: {
    CampaignPreview,
    TemplateForm,
    EmptyPlaceholder,
  },

  data() {
    return {
      curItem: null,
      isEditing: false,
      isFormVisible: false,
      previewItem: null,
    };
  },

  methods: {
    fetchTemplates() {
      this.$api.getTemplates();
    },

    // Show the edit form.
    showEditForm(data) {
      this.curItem = data;
      this.isFormVisible = true;
      this.isEditing = true;
    },

    // Show the new form.
    showNewForm() {
      this.curItem = { type: 'campaign' };
      this.isFormVisible = true;
      this.isEditing = false;
    },

    formFinished() {
      this.$api.getTemplates();
    },

    previewTemplate(c) {
      this.previewItem = c;
    },

    closePreview() {
      this.previewItem = null;
    },

    cloneTemplate(name, t) {
      const data = {
        name,
        type: t.type,
        subject: t.subject,
        body: t.body,
        body_source: t.bodySource,
      };
      this.$api.createTemplate(data).then((d) => {
        this.$api.getTemplates();
        this.$emit('finished');
        this.$utils.toast(`'${d.name}' created`);
      });
    },

    makeTemplateDefault(tpl) {
      this.$api.makeTemplateDefault(tpl.id).then(() => {
        this.$api.getTemplates();
        this.$utils.toast(this.$t('globals.messages.created', { name: tpl.name }));
      });
    },

    deleteTemplate(tpl) {
      this.$api.deleteTemplate(tpl.id).then(() => {
        this.$api.getTemplates();
        this.$utils.toast(this.$t('globals.messages.deleted', { name: tpl.name }));
      });
    },
  },

  computed: {
    ...mapState(['templates', 'loading']),
  },

  created() {
    this.$root.$on('page.refresh', this.fetchTemplates);
  },

  destroyed() {
    this.$root.$off('page.refresh', this.fetchTemplates);
  },

  mounted() {
    this.$api.getTemplates();
  },
});
</script>
