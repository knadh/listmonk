<template>
  <section class="campaign">
    <header class="columns page-header is-mobile is-vcentered mb-4">
      <div class="column is-6">
        <h4 v-if="isEditing" class="title is-4 mb-0">
          {{ data.name }}
        </h4>
        <h4 v-else class="title is-4 mb-0">
          {{ $t('campaigns.newCampaign') }}
        </h4>
        <p v-if="isEditing && data.status" class="tags mt-1 mb-0">
          <b-tag v-if="isEditing" :class="data.status" size="is-small">
            {{ $t(`campaigns.status.${data.status}`) }}
          </b-tag>
          <span v-if="isEditing" class="has-text-grey-light is-size-7 ml-1">
            #{{ data.id }}
          </span>
        </p>
      </div>

      <div class="column is-6 has-text-right">
        <div v-if="canManage" class="buttons is-right">
          <b-button v-if="isEditing && canEdit" @click="() => onSubmit('update')" :loading="loading.campaigns"
            type="is-primary" icon-left="content-save-outline" data-cy="btn-save" />

          <b-dropdown v-if="isEditing && (canStart || canSchedule || canUnSchedule)" position="is-bottom-left"
            append-to-body>
            <template #trigger>
              <b-button type="is-primary" icon-right="chevron-down">
                {{ $t('globals.buttons.actions') }}
              </b-button>
            </template>

            <b-dropdown-item v-if="canStart" @click="startCampaign">
              <b-icon icon="rocket-launch-outline" /> {{ $t('campaigns.start') }}
            </b-dropdown-item>
            <b-dropdown-item v-if="canSchedule" @click="startCampaign">
              <b-icon icon="clock-start" /> {{ $t('campaigns.schedule') }}
            </b-dropdown-item>
            <b-dropdown-item v-if="canUnSchedule" @click="$utils.confirm(null, unscheduleCampaign)">
              <b-icon icon="clock-start" /> {{ $t('campaigns.unSchedule') }}
            </b-dropdown-item>
          </b-dropdown>
        </div>
      </div>
    </header>

    <b-loading :active="loading.campaigns" />

    <b-tabs type="is-boxed" :animated="false" v-model="activeTab" @input="onTab">
      <b-tab-item :label="$tc('globals.terms.campaign')" label-position="on-border" value="campaign"
        icon="rocket-launch-outline">
        <section class="wrap">
          <div class="columns is-multiline">
            <div class="column is-7-desktop is-12-tablet">
              <form @submit.prevent="() => onSubmit(isNew ? 'create' : 'update')">
                <b-field :label="$t('globals.fields.name')" label-position="on-border">
                  <b-input :maxlength="200" :ref="'focus'" v-model="form.name" name="name" :disabled="!canEdit"
                    :placeholder="$t('globals.fields.name')" required autofocus />
                </b-field>

                <b-field :label="$t('campaigns.subject')" label-position="on-border">
                  <b-input :maxlength="5000" v-model="form.subject" name="subject" :disabled="!canEdit"
                    :placeholder="$t('campaigns.subject')" required />
                </b-field>

                <b-field :label="$t('campaigns.fromAddress')" label-position="on-border">
                  <b-input :maxlength="200" v-model="form.fromEmail" name="from_email" :disabled="!canEdit"
                    :placeholder="$t('campaigns.fromAddressPlaceholder')" required />
                </b-field>

                <list-selector v-model="form.lists" :selected="form.lists" :all="lists.results" :disabled="!canEdit"
                  :label="$t('globals.terms.lists')" :placeholder="$t('campaigns.sendToLists')" />

                <div class="columns is-multiline">
                  <div class="column is-6-desktop is-12-tablet">
                    <b-field :label="$tc('globals.terms.messenger')" label-position="on-border">
                      <b-select :placeholder="$tc('globals.terms.messenger')" v-model="form.messenger" name="messenger"
                        :disabled="!canEdit" required expanded>
                        <template v-if="emailMessengers.length > 1">
                          <optgroup label="email">
                            <option v-for="m in emailMessengers" :value="m" :key="m">
                              {{ m }}
                            </option>
                          </optgroup>
                        </template>
                        <template v-else>
                          <option value="email">email</option>
                        </template>
                        <option v-for="m in otherMessengers" :value="m" :key="m">{{ m }}</option>
                      </b-select>
                    </b-field>
                  </div>
                  <div class="column is-6-desktop is-12-tablet">
                    <b-field :label="$t('campaigns.format')" label-position="on-border" class="mb-0">
                      <b-select v-model="form.content.contentType" :disabled="!canEdit || isEditing" value="richtext"
                        expanded>
                        <option v-for="(name, f) in contentTypes" :key="f" name="format" :value="f"
                          :data-cy="`check-${f}`">
                          {{ name }}
                        </option>
                      </b-select>
                    </b-field>
                  </div>
                </div>

                <b-field :label="$t('globals.terms.tags')" label-position="on-border">
                  <b-taginput v-model="form.tags" name="tags" :disabled="!canEdit" ellipsis icon="tag-outline"
                    :placeholder="$t('globals.terms.tags')" />
                </b-field>
                <hr />

                <div class="columns is-multiline">
                  <div class="column is-4-desktop is-12-tablet">
                    <b-field :label="$t('campaigns.sendLater')" data-cy="btn-send-later">
                      <b-switch v-model="form.sendLater" :disabled="!canEdit" />
                    </b-field>
                  </div>
                  <div class="column is-8-desktop is-12-tablet">
                    <b-field v-if="form.sendLater" data-cy="send_at"
                      :message="form.sendAtDate ? $utils.duration(Date(), form.sendAtDate) : ''">
                      <b-datetimepicker v-model="form.sendAtDate" :disabled="!canEdit" required editable mobile-native
                        position="is-top-right" :placeholder="$t('campaigns.dateAndTime')" icon="calendar-clock"
                        :timepicker="{ hourFormat: '24' }" :datetime-formatter="formatDateTime"
                        horizontal-time-picker expanded />
                    </b-field>
                  </div>
                </div>

                <div class="mt-4">
                  <p class="has-text-right">
                    <a href="#" @click.prevent="onShowHeaders" data-cy="btn-headers">
                      <b-icon icon="plus" />{{ $t('settings.smtp.setCustomHeaders') }}
                    </a>
                  </p>
                  <b-field v-if="form.headersStr !== '[]' || isHeadersVisible" label-position="on-border"
                    :message="$t('campaigns.customHeadersHelp')">
                    <b-input v-model="form.headersStr" name="headers" type="textarea"
                      placeholder="[{&quot;X-Custom&quot;: &quot;value&quot;}, {&quot;X-Custom2&quot;: &quot;value&quot;}]"
                      :disabled="!canEdit" />
                  </b-field>
                </div>
                <hr />

                <b-field v-if="isNew">
                  <b-button native-type="submit" type="is-primary" :loading="loading.campaigns" data-cy="btn-continue">
                    {{ $t('campaigns.continue') }}
                  </b-button>
                </b-field>
              </form>
            </div>
            <div v-if="canManage" class="column is-4-desktop is-offset-1-desktop is-12-tablet">
              <div class="box">
                <h3 class="title is-size-6">
                  {{ $t('campaigns.sendTest') }}
                </h3>
                <b-field :message="$t('campaigns.sendTestHelp')">
                  <b-taginput v-model="form.testEmails" :before-adding="$utils.validateEmail" :disabled="isNew" ellipsis
                    icon="email-outline" :placeholder="$t('campaigns.testEmails')" />
                </b-field>
                <b-field>
                  <b-button @click="() => onSubmit('test')" :loading="loading.campaigns" :disabled="isNew"
                    type="is-primary" icon-left="email-outline" expanded>
                    {{ $t('campaigns.send') }}
                  </b-button>
                </b-field>
              </div>
            </div>
          </div>
        </section>
      </b-tab-item><!-- campaign -->

      <b-tab-item :label="$t('campaigns.content')" icon="text" :disabled="isNew" value="content">
        <editor v-if="data.id" v-model="form.content" :id="data.id" :title="data.name" :disabled="!canEdit"
          :templates="templates" :content-types="contentTypes" :is-mobile="isMobile" />

        <div class="columns is-multiline mt-4">
          <div class="column is-6-desktop is-12-tablet">
            <p v-if="!isAttachFieldVisible" class="is-size-6 has-text-grey">
              <a href="#" @click.prevent="onShowAttachField()" data-cy="btn-attach">
                <b-icon icon="file-upload-outline" size="is-small" />
                {{ $t('campaigns.addAttachments') }}
              </a>
            </p>

            <b-field v-if="isAttachFieldVisible" :label="$t('campaigns.attachments')" label-position="on-border"
              expanded data-cy="media">
              <b-taginput v-model="form.media" name="media" ellipsis icon="tag-outline" ref="media" field="filename"
                @focus="onOpenAttach" :disabled="!canEdit" />
            </b-field>
          </div>
          <div class="column is-6-desktop is-12-tablet has-text-right-desktop has-text-left-tablet">
            <a href="https://listmonk.app/docs/templating/#template-expressions" target="_blank"
              rel="noopener noreferer">
              <b-icon icon="code" /> {{ $t('campaigns.templatingRef') }}</a>
            <div v-if="canEdit && form.content.contentType !== 'plain'" class="is-size-6 has-text-grey mt-2">
              <a v-if="form.altbody === null" href="#" @click.prevent="onAddAltBody">
                <b-icon icon="text" size="is-small" /> {{ $t('campaigns.addAltText') }}
              </a>
              <a v-else href="#" @click.prevent="$utils.confirm(null, onRemoveAltBody)">
                <b-icon icon="trash-can-outline" size="is-small" />
                {{ $t('campaigns.removeAltText') }}
              </a>
            </div>
          </div>
        </div>

        <div v-if="canEdit && form.content.contentType !== 'plain'" class="alt-body">
          <b-input v-if="form.altbody !== null" v-model="form.altbody" type="textarea" :disabled="!canEdit" />
        </div>
      </b-tab-item><!-- content -->

      <b-tab-item :label="$t('globals.terms.attribs')" icon="code" value="attribs" :disabled="isNew">
        <section class="wrap">
          <b-field :label="$t('globals.terms.attribs')" :message="$t('campaigns.attribsHelp')"
            label-position="on-border">
            <b-input v-model="form.attribsStr" type="textarea" :disabled="!canEdit" rows="15" />
          </b-field>
        </section>
      </b-tab-item><!-- attribs -->

      <b-tab-item :label="$t('campaigns.archive')" icon="newspaper-variant-outline" value="archive" :disabled="isNew">
        <section class="wrap">
          <div class="columns">
            <div class="column is-4">
              <b-field :label="$t('campaigns.archiveEnable')" data-cy="btn-archive"
                :message="$t('campaigns.archiveHelp')">
                <div class="columns">
                  <div class="column">
                    <b-switch data-cy="btn-archive" v-model="form.archive" :disabled="!canArchive" />
                  </div>
                  <div class="column is-12">
                    <a :href="`${serverConfig.root_url}/archive/${data.uuid}`" target="_blank" rel="noopener noreferer"
                      :class="{ 'has-text-grey-light': !form.archive }" aria-label="$t('campaigns.archive')">
                      <b-icon icon="link-variant" />
                    </a>
                  </div>
                </div>
              </b-field>
            </div>
            <div class="column is-8">
              <b-field grouped position="is-right">
                <b-field v-if="!canEdit && canArchive">
                  <b-button @click="onUpdateCampaignArchive" :loading="loading.campaigns" type="is-primary"
                    icon-left="content-save-outline" data-cy="btn-save">
                    {{ $t('globals.buttons.saveChanges') }}
                  </b-button>
                </b-field>
              </b-field>
            </div>
          </div>

          <div class="columns">
            <div class="column is-6">
              <b-field :label="$tc('globals.terms.template')" label-position="on-border">
                <b-select :placeholder="$tc('globals.terms.template')" v-model="form.archiveTemplateId" name="template"
                  :disabled="!canArchive || !form.archive || form.content.contentType === 'visual'" required>
                  <template v-for="t in templates">
                    <option v-if="t.type === 'campaign'" :value="t.id" :key="t.id">
                      {{ t.name }}
                    </option>
                  </template>
                </b-select>
              </b-field>
            </div>

            <div class="column is-6">
              <b-field grouped position="is-right">
                <b-field v-if="form.archive && (!this.form.archiveMetaStr || this.form.archiveMetaStr === '{}')">
                  <a class="button is-primary" href="#" @click.prevent="onFillArchiveMeta" aria-label="{}"><b-icon
                      icon="code" /></a>
                </b-field>
                <b-field v-if="form.archive">
                  <b-button @click="onToggleArchivePreview" type="is-primary" icon-left="file-find-outline"
                    data-cy="btn-preview">
                    {{ $t('campaigns.preview') }}
                  </b-button>
                </b-field>
              </b-field>
            </div>
          </div>
          <b-field>
            <b-field :label="$t('campaigns.archiveSlug')" label-position="on-border"
              :message="$t('campaigns.archiveSlugHelp')">
              <b-input :maxlength="200" :ref="'focus'" v-model="form.archiveSlug" name="archive_slug"
                data-cy="archive-slug" :disabled="!canArchive || !form.archive" />
            </b-field>
          </b-field>
          <b-field :label="$t('campaigns.archiveMeta')" :message="$t('campaigns.archiveMetaHelp')"
            label-position="on-border">
            <b-input v-model="form.archiveMetaStr" name="archive_meta" type="textarea" data-cy="archive-meta"
              :disabled="!canArchive || !form.archive" rows="20" />
          </b-field>
        </section>
      </b-tab-item><!-- archive -->
    </b-tabs>

    <b-modal scroll="keep" :aria-modal="true" :active.sync="isAttachModalOpen" :width="900">
      <div class="modal-card content" style="width: auto">
        <section expanded class="modal-card-body">
          <media is-modal @selected="onAttachSelect" />
        </section>
      </div>
    </b-modal>

    <campaign-preview v-if="isPreviewingArchive" @close="onToggleArchivePreview" type="campaign" :id="data.id"
      :archive-meta="form.archiveMetaStr" :title="data.title" :content-type="data.contentType"
      :template-id="form.archiveTemplateId" is-post is-archive />
  </section>
</template>

<script>
import dayjs from 'dayjs';
import htmlToPlainText from 'textversionjs';
import Vue from 'vue';
import { mapState } from 'vuex';

import CampaignPreview from '../components/CampaignPreview.vue';
import Editor from '../components/Editor.vue';
import ListSelector from '../components/ListSelector.vue';
import Media from './Media.vue';

export default Vue.extend({
  components: {
    ListSelector,
    Editor,
    Media,
    CampaignPreview,
  },

  data() {
    return {
      saveStatus: '',

      contentTypes: Object.freeze({
        richtext: this.$t('campaigns.richText'),
        html: this.$t('campaigns.rawHTML'),
        markdown: this.$t('campaigns.markdown'),
        emailmd: this.$t('campaigns.emailmd'),
        plain: this.$t('campaigns.plainText'),
        visual: this.$t('campaigns.visual'),
      }),

      isNew: false,
      isEditing: false,
      isHeadersVisible: false,
      isAttachFieldVisible: false,
      isAttachModalOpen: false,
      isPreviewingArchive: false,
      activeTab: 'campaign',

      data: {},

      // IDs from ?list_id query param.
      selListIDs: [],

      // Binds form input values.
      form: {
        archiveSlug: null,
        name: '',
        subject: '',
        fromEmail: '',
        headersStr: '[]',
        headers: [],
        attribsStr: '{}',
        messenger: 'email',
        lists: [],
        tags: [],
        sendAt: null,
        content: {
          contentType: 'richtext',
          body: '',
          bodySource: null,
          templateId: null,
        },
        altbody: null,
        media: [],

        // Parsed Date() version of send_at from the API.
        sendAtDate: null,
        sendLater: false,
        archive: false,
        archiveMetaStr: '{}',
        archiveMeta: {},
        testEmails: [],
      },

      // UI state
      isMobile: window.innerWidth <= 768,
      isEditorModalOpen: false,
    };
  },

  methods: {
    formatDateTime(s) {
      return dayjs(s).format('YYYY-MM-DD HH:mm');
    },

    onToggleArchivePreview() {
      this.isPreviewingArchive = !this.isPreviewingArchive;
    },

    onAddAltBody() {
      this.form.altbody = htmlToPlainText(this.form.content.body);
    },

    onRemoveAltBody() {
      this.form.altbody = null;
    },

    onShowHeaders() {
      this.isHeadersVisible = !this.isHeadersVisible;
    },

    onShowAttachField() {
      this.isAttachFieldVisible = true;
      this.$nextTick(() => {
        this.$refs.media.focus();
      });
    },

    onOpenAttach() {
      this.isAttachModalOpen = true;
    },

    onAttachSelect(o) {
      if (this.form.media.some((m) => m.id === o.id)) {
        return;
      }

      this.form.media.push(o);
    },

    isUnsaved() {
      return this.data.body !== this.form.content.body
        || this.data.contentType !== this.form.content.contentType;
    },

    onTab(tab) {
      // this.$router.replace({ hash: `#${tab}` });
      window.history.replaceState({}, '', `#${tab}`);
    },

    onFillArchiveMeta() {
      const archiveStr = `{"email": "email@domain.com", "name": "${this.$t('globals.fields.name')}", "attribs": {}}`;
      this.form.archiveMetaStr = this.$utils.getPref('campaign.archiveMetaStr') || JSON.stringify(JSON.parse(archiveStr), null, 4);
    },

    onSubmit(typ) {
      // Validate custom JSON headers.
      if (this.form.headersStr && this.form.headersStr !== '[]') {
        try {
          this.form.headers = JSON.parse(this.form.headersStr);
        } catch (e) {
          this.$utils.toast(e.toString(), 'is-danger');
          return;
        }
      } else {
        this.form.headers = [];
      }

      // Validate archive JSON body.
      if (this.form.archive && this.form.archiveMetaStr) {
        try {
          this.form.archiveMeta = JSON.parse(this.form.archiveMetaStr);
        } catch (e) {
          this.$utils.toast(e.toString(), 'is-danger');
          return;
        }
      }

      // Validate custom JSON attribs.
      let attribs = null;
      if (this.form.attribsStr && this.form.attribsStr.trim()) {
        try {
          attribs = JSON.parse(this.form.attribsStr);
        } catch (e) {
          this.$utils.toast(
            `${this.$t('subscribers.invalidJSON')}: ${e.toString()}`,
            'is-danger',

            3000,
          );
          return;
        }
      }
      this.form.attribs = attribs;

      switch (typ) {
        case 'create':
          this.createCampaign();
          break;
        case 'test':
          this.sendTest();
          break;
        default:
          this.updateCampaign();
          break;
      }
    },

    getCampaign(id) {
      return this.$api.getCampaign(id).then((data) => {
        this.data = data;
        this.form = {
          ...this.form,
          ...data,
          headersStr: JSON.stringify(data.headers, null, 4),
          archiveMetaStr: data.archiveMeta ? JSON.stringify(data.archiveMeta, null, 4) : '{}',
          attribsStr: data.attribs ? JSON.stringify(data.attribs, null, 4) : '{}',

          // The structure that is populated by editor input event.
          content: {
            contentType: data.contentType,
            body: data.body,
            bodySource: data.bodySource,
            templateId: data.templateId,
          },
        };
        this.isAttachFieldVisible = this.form.media.length > 0;

        this.form.media = this.form.media.map((f) => {
          if (!f.id) {
            return { ...f, filename: `❌ ${f.filename}` };
          }
          return f;
        });
      });
    },

    sendTest() {
      const data = {
        id: this.data.id,
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.fromEmail,
        messenger: this.form.messenger,
        type: 'regular',
        headers: this.form.headers,
        tags: this.form.tags,
        template_id: this.form.content.templateId,
        content_type: this.form.content.contentType,
        body: this.form.content.body,
        altbody: this.form.content.contentType !== 'plain' ? this.form.altbody : null,
        subscribers: this.form.testEmails,
        media: this.form.media.map((m) => m.id),
      };

      this.$api.testCampaign(data).then(() => {
        this.$utils.toast(this.$t('campaigns.testSent'));
      });
      return false;
    },

    createCampaign() {
      const data = {
        archiveSlug: this.form.subject,
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.fromEmail,
        content_type: this.form.content.contentType,
        messenger: this.form.messenger,
        type: 'regular',
        tags: this.form.tags,
        send_at: this.form.sendLater ? this.form.sendAtDate : null,
        headers: this.form.headers,
        attribs: this.form.attribs,
        media: this.form.media.map((m) => m.id),
      };

      this.$api.createCampaign(data).then((d) => {
        this.$router.push({ name: 'campaign', hash: '#content', params: { id: d.id } });
      });
      return false;
    },

    async updateCampaign(typ) {
      const data = {
        archive_slug: this.form.archiveSlug,
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.fromEmail,
        messenger: this.form.messenger,
        type: 'regular',
        tags: this.form.tags,
        send_at: this.form.sendLater ? this.form.sendAtDate : null,
        headers: this.form.headers,
        attribs: this.form.attribs,
        template_id: this.form.content.templateId,
        content_type: this.form.content.contentType,
        body: this.form.content.body,
        body_source: this.form.content.bodySource,
        altbody: this.form.content.contentType !== 'plain' ? this.form.altbody : null,
        archive: this.form.archive,
        archive_template_id: this.form.archiveTemplateId,
        archive_meta: this.form.archiveMeta,
        media: this.form.media.map((m) => m.id),
      };

      let typMsg = 'globals.messages.updated';
      if (typ === 'start') {
        typMsg = 'campaigns.started';
      }

      if (!this.form.sendAtDate) {
        this.form.sendLater = false;
      }

      // This promise is used by startCampaign to first save before starting.
      return new Promise((resolve) => {
        this.$api.updateCampaign(this.data.id, data).then((d) => {
          this.data = d;
          this.form.archiveSlug = d.archiveSlug;
          this.form.attribsStr = d.attribs ? JSON.stringify(d.attribs, null, 4) : '{}';

          if (typ !== 'silent') {
            this.$utils.toast(this.$t(typMsg, { name: d.name }));
          }
          resolve();
        });
      });
    },

    debounce(func, wait) {
      let timeout;
      return function debounced(...args) {
        const context = this;
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(context, args), wait);
      };
    },

    autoSave() {
      // Placeholder, overwritten in mounted
    },

    onUpdateCampaignArchive() {
      if (this.isEditing && this.canEdit) {
        return;
      }

      const data = {
        archive: this.form.archive,
        archive_template_id: this.form.archiveTemplateId,
        archive_meta: JSON.parse(this.form.archiveMetaStr),
        archive_slug: this.form.archiveSlug,
      };

      this.$api.updateCampaignArchive(this.data.id, data).then((d) => {
        this.form.archiveSlug = d.archiveSlug;
      });
    },

    // Starts or schedule a campaign.
    startCampaign() {
      if (!this.canStart && !this.canSchedule) {
        return;
      }

      this.$utils.confirm(
        null,
        () => {
          // First save the campaign.
          this.updateCampaign().then(() => {
            // Then start/schedule it.
            let status = '';
            if (this.canStart) {
              status = 'running';
            } else if (this.canSchedule) {
              status = 'scheduled';
            } else {
              return;
            }

            this.$api.changeCampaignStatus(this.data.id, status).then(() => {
              this.$router.push({ name: 'campaigns' });
            });
          });
        },
      );
    },

    unscheduleCampaign() {
      this.$api.changeCampaignStatus(this.data.id, 'draft').then((d) => {
        this.data = d;
      });
    },

    onResize() {
      this.isMobile = window.innerWidth <= 768;
    },

    toggleEditorModal() {
      this.isEditorModalOpen = !this.isEditorModalOpen;
    },
  },

  computed: {
    ...mapState(['serverConfig', 'loading', 'lists', 'templates']),

    canManage() {
      return this.$can('campaigns:manage_all', 'campaigns:manage');
    },

    canEdit() {
      return this.isNew
        || this.data.status === 'draft' || this.data.status === 'scheduled' || this.data.status === 'paused';
    },

    canSchedule() {
      return (this.data.status === 'draft' || this.data.status === 'paused') && (this.form.sendLater && this.form.sendAtDate);
    },

    canUnSchedule() {
      return this.data.status === 'scheduled';
    },

    canStart() {
      return (this.data.status === 'draft' || this.data.status === 'paused') && !this.form.sendLater;
    },

    canArchive() {
      return this.data.status !== 'cancelled' && this.data.type !== 'optin';
    },

    selectedLists() {
      if (this.selListIDs.length === 0 || !this.lists.results) {
        return [];
      }

      return this.lists.results.filter((l) => this.selListIDs.indexOf(l.id) > -1);
    },

    emailMessengers() {
      return ['email', ...this.serverConfig.messengers.filter((m) => m.startsWith('email-'))];
    },

    otherMessengers() {
      return this.serverConfig.messengers.filter((m) => m !== 'email' && !m.startsWith('email-'));
    },
  },

  beforeDestroy() {
    window.removeEventListener('resize', this.onResize);
    this.$events.$off('campaign.update');
  },

  beforeRouteLeave(to, from, next) {
    if (this.isUnsaved()) {
      this.$utils.confirm(this.$t('globals.messages.confirmDiscard'), () => next(true));
      return;
    }
    next(true);
  },

  watch: {
    selectedLists() {
      this.form.lists = this.selectedLists;
    },

    'data.sendAt': function onSendAtChange() {
      if (this.data.sendAt !== null) {
        this.form.sendLater = true;
        this.form.sendAtDate = dayjs(this.data.sendAt).toDate();
      } else {
        this.form.sendLater = false;
        this.form.sendAtDate = null;
      }
    },

    'form.content': {
      handler() {
        if (!this.isEditing || !this.data.id) {
          return;
        }

        this.saveStatus = 'saving';
        this.autoSave();
      },
      deep: true,
    },
  },

  mounted() {
    this.autoSave = this.debounce(() => {
      this.updateCampaign('silent').then(() => {
        this.saveStatus = 'saved';
        setTimeout(() => {
          this.saveStatus = '';
        }, 2000);
      });
    }, 2000);

    window.addEventListener('resize', this.onResize);
    window.onbeforeunload = () => this.isUnsaved() || null;

    // Fill default form fields.
    this.form.fromEmail = this.serverConfig.from_email;

    // New campaign.
    const { id } = this.$route.params;
    if (id === 'new') {
      this.isNew = true;

      if (this.$route.query.list_id) {
        // Multiple list_id query params.
        let strIds = [];
        if (typeof this.$route.query.list_id === 'object') {
          strIds = this.$route.query.list_id;
        } else {
          strIds = [this.$route.query.list_id];
        }

        this.selListIDs = strIds.map((v) => parseInt(v, 10));
      }
    } else {
      const intID = parseInt(id, 10);
      if (intID <= 0 || Number.isNaN(intID)) {
        this.$utils.toast(this.$t('campaigns.invalid'));
        return;
      }

      this.isEditing = true;
    }

    // Get templates list.
    this.$api.getTemplates().then((data) => {
      if (data.length > 0) {
        if (!this.form.templateId) {
          const tpl = data.find((i) => i.isDefault === true);
          this.form.templateId = tpl.id;
        }
      }
    });

    // Fetch campaign.
    if (this.isEditing) {
      this.getCampaign(id).then(() => {
        if (this.$route.hash !== '') {
          this.activeTab = this.$route.hash.replace('#', '');
        }
      });
    } else {
      this.form.messenger = 'email';
    }

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });

    this.$events.$on('campaign.update', () => {
      this.onSubmit('update');
    });
  },

});
</script>
