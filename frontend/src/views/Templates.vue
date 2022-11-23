<template>
  <section class="templates">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">{{ $t('globals.terms.templates') }}
          <span v-if="templates.length > 0">({{ templates.length }})</span></h1>
      </div>
      <div class="column has-text-right">
        <b-field expanded>
          <b-button expanded type="is-primary" icon-left="plus" class="btn-new"
            @click="showNewForm">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <b-table :data="templates" :hoverable="true" :loading="loading.templates"
      default-sort="createdAt">
      <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')"
        :td-attrs="$utils.tdID" sortable>
        <a href="#" @click.prevent="showEditForm(props.row)">
          {{ props.row.name }}
        </a>
        <b-tag v-if="props.row.isDefault">{{ $t('templates.default') }}</b-tag>

        <p class="is-size-7 has-text-grey" v-if="props.row.type === 'tx'">
          {{ props.row.subject }}
          </p>
      </b-table-column>

      <b-table-column v-slot="props" field="type"
        :label="$t('globals.fields.type')" sortable>
        <b-tag v-if="props.row.type === 'campaign'"
          :class="props.row.type" :data-cy="`type-${props.row.type}`">
          {{ $tc('globals.terms.campaign', 1) }}
        </b-tag>
        <b-tag v-else
          :class="props.row.type" :data-cy="`type-${props.row.type}`">
          {{ $tc('globals.terms.tx', 1) }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="id" :label="$t('globals.fields.id')" sortable>
        {{ props.row.id }}
      </b-table-column>

      <b-table-column v-slot="props" field="createdAt"
        :label="$t('globals.fields.createdAt')" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </b-table-column>

      <b-table-column v-slot="props" field="updatedAt"
        :label="$t('globals.fields.updatedAt')" sortable>
        {{ $utils.niceDate(props.row.updatedAt) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a href="#" @click.prevent="previewTemplate(props.row)" data-cy="btn-preview">
            <b-tooltip :label="$t('templates.preview')" type="is-dark">
              <b-icon icon="file-find-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="#" @click.prevent="showEditForm(props.row)" data-cy="btn-edit">
            <b-tooltip :label="$t('globals.buttons.edit')" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a href="" @click.prevent="$utils.prompt(`Clone template`,
              { placeholder: 'Name', value: `Copy of ${props.row.name}`},
              (name) => cloneTemplate(name, props.row))"
              data-cy="btn-clone">
            <b-tooltip :label="$t('globals.buttons.clone')" type="is-dark">
              <b-icon icon="file-multiple-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a v-if="!props.row.isDefault && props.row.type !== 'tx'" href="#"
            @click.prevent="$utils.confirm(null, () => makeTemplateDefault(props.row))"
            data-cy="btn-set-default">
            <b-tooltip :label="$t('templates.makeDefault')" type="is-dark">
              <b-icon icon="check-circle-outline" size="is-small" />
            </b-tooltip>
          </a>
          <span v-else class="a has-text-grey-light">
              <b-icon icon="check-circle-outline" size="is-small" />
          </span>

          <a v-if="!props.row.isDefault" href="#"
            @click.prevent="$utils.confirm(null, () => deleteTemplate(props.row))"
            data-cy="btn-delete">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
          <span v-else class="a has-text-grey-light">
              <b-icon icon="trash-can-outline" size="is-small" />
          </span>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.templates">
        <empty-placeholder />
      </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible"
      :width="1200" :can-cancel="false" class="template-modal">
      <template-form :data="curItem" :isEditing="isEditing"
        @finished="formFinished"></template-form>
    </b-modal>

    <campaign-preview v-if="previewItem"
      type='template'
      :id="previewItem.id"
      :templateType="previewItem.type"
      :title="previewItem.name"
      @close="closePreview"></campaign-preview>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import TemplateForm from './TemplateForm.vue';
import CampaignPreview from '../components/CampaignPreview.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

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

  mounted() {
    this.$api.getTemplates();
  },
});
</script>
