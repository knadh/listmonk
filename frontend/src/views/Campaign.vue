<template>
  <section class="campaign">
    <header class="row page-header">
      <div class="col-8">
        <p v-if="isEditing && data.status" class="hstack">
          <oat-badge v-if="isEditing" :type="data.status">
            {{ $t(`campaigns.status.${data.status}`) }}
          </oat-badge>
          <oat-badge v-if="data.type === 'optin'" :type="data.type">
            {{ $t('lists.optin') }}
          </oat-badge>
          <span v-if="isEditing" class="text-lighter text-7 " :data-campaign-id="data.id">
            {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
            {{ $t('globals.fields.uuid') }}: <copy-text :text="data.uuid" />
          </span>
        </p>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ $t('campaigns.newCampaign') }}
        </h4>
      </div>

      <div class="col-4 col-end align-right">
        <div v-if="canManage || canSend" class="hstack justify-end">
          <oat-field v-if="isEditing && canEdit">
            <oat-field v-if="canManage">
              <button type="button" @click="() => onSubmit('update')" :loading="loading.campaigns" data-variant="primary"
                data-cy="btn-save" aria-keyshortcuts="ctrl+s">
                <span class="has-kbd">{{ $t('globals.buttons.saveChanges') }} <span class="kbd">Ctrl+S</span></span>
              </button>
            </oat-field>
            <oat-field v-if="canSend && canStart">
              <button type="button" @click="startCampaign" :loading="loading.campaigns" data-variant="primary"
                icon-left="rocket-launch-outline" data-cy="btn-start">
                {{ $t('campaigns.start') }}
              </button>
            </oat-field>
            <oat-field v-if="canSend && canSchedule">
              <button type="button" @click="startCampaign" :loading="loading.campaigns" data-variant="primary"
                icon-left="clock-start" data-cy="btn-schedule">
                {{ $t('campaigns.schedule') }}
              </button>
            </oat-field>
            <oat-field v-if="canSend && canUnSchedule">
              <button type="button" class="outline" @click="$utils.confirm(null, unscheduleCampaign)"
                :loading="loading.campaigns" icon-left="clock-start" data-cy="btn-unschedule">
                {{ $t('campaigns.unSchedule') }}
              </button>
            </oat-field>
          </oat-field>
        </div>
      </div>
    </header>

    <oat-loading :active="loading.campaigns" />

    <oat-tabs v-model="activeTab" @input="onTab">
      <oat-tab-item :label="$tc('globals.terms.campaign')" value="campaign"
        icon="rocket-launch-outline">
        <section class="wrap">
          <div class="row">
            <div class="col-7">
              <form @submit.prevent="() => onSubmit(isNew ? 'create' : 'update')">
                <oat-field :label="$t('globals.fields.name')">
                  <input aria-label="field" :maxlength="200" :ref="'focus'" v-model="form.name" name="name" :disabled="!canEdit"
                    :placeholder="$t('globals.fields.name')" required>
                </oat-field>

                <oat-field :label="$t('campaigns.subject')">
                  <input aria-label="field" :maxlength="5000" v-model="form.subject" name="subject" :disabled="!canEdit"
                    :placeholder="$t('campaigns.subject')" required>
                </oat-field>

                <oat-field :label="$t('campaigns.fromAddress')">
                  <input aria-label="field" :maxlength="200" v-model="form.fromEmail" name="from_email" :disabled="!canEdit"
                    :placeholder="$t('campaigns.fromAddressPlaceholder')" required>
                </oat-field>

                <list-selector v-model="form.lists" :selected="form.lists" :all="lists.results" :disabled="!canEdit"
                  :label="$t('globals.terms.lists')" :placeholder="$t('campaigns.sendToLists')" />

                <div class="row">
                  <div class="col-6">
                    <oat-field :label="$tc('globals.terms.messenger')">
                      <select aria-label="field" :placeholder="$tc('globals.terms.messenger')" v-model="form.messenger" name="messenger"
                        :disabled="!canEdit" required>
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
                      </select>
                    </oat-field>
                  </div>
                  <div class="col-6">
                    <oat-field :label="$t('campaigns.format')" class="mr-4 mb-0">
                      <select aria-label="field" v-model="form.content.contentType" :disabled="!canEdit || isEditing" value="richtext"
                       >
                        <option v-for="(name, f) in contentTypes" :key="f" name="format" :value="f"
                          :data-cy="`check-${f}`">
                          {{ name }}
                        </option>
                      </select>
                    </oat-field>
                  </div>
                </div>

                <oat-field :label="$t('globals.terms.tags')">
                  <oat-tag-input v-model="form.tags" name="tags" :disabled="!canEdit"
                    :placeholder="$t('globals.terms.tags')" />
                </oat-field>
                <hr />

                <div class="row">
                  <div class="col-4">
                    <oat-field :label="$t('campaigns.sendLater')" data-cy="btn-send-later">
                      <oat-switch v-model="form.sendLater" :disabled="!canEdit" />
                    </oat-field>
                  </div>
                  <div class="col-12">
                    <br />
                    <oat-field v-if="form.sendLater" data-cy="send_at"
                      :message="form.sendAtDate ? $utils.duration(Date(), form.sendAtDate) : ''">
                      <oat-date-input datetime v-model="form.sendAtDate" :disabled="!canEdit" required editable mobile-native
 :placeholder="$t('campaigns.dateAndTime')"
                        horizontal-time-picker />
                    </oat-field>
                  </div>
                </div>

                <div>
                  <p class="align-right">
                    <a href="#" @click.prevent="onShowHeaders" data-cy="btn-headers">
                      <oat-icon icon="plus" />{{ $t('settings.smtp.setCustomHeaders') }}
                    </a>
                  </p>
                  <oat-field v-if="form.headersStr !== '[]' || isHeadersVisible"
                    :message="$t('campaigns.customHeadersHelp')">
                    <textarea aria-label="field" v-model="form.headersStr" name="headers"
                      placeholder="[{&quot;X-Custom&quot;: &quot;value&quot;}, {&quot;X-Custom2&quot;: &quot;value&quot;}]"
                      :disabled="!canEdit" />
                  </oat-field>
                </div>
                <hr />

                <oat-field v-if="isNew">
                  <button type="submit" data-variant="primary" :loading="loading.campaigns" data-cy="btn-continue">
                    {{ $t('campaigns.continue') }}
                  </button>
                </oat-field>
              </form>
            </div>
            <div v-if="canManage" class="col-4 offset-1">
              <br />
              <div class="card">
                <h3>
                  {{ $t('campaigns.sendTest') }}
                </h3>
                <oat-field :message="$t('campaigns.sendTestHelp')">
                  <oat-tag-input v-model="form.testEmails" :before-adding="$utils.validateEmail" :disabled="isNew"
                    icon="email-outline" :placeholder="$t('campaigns.testEmails')" />
                </oat-field>
                <oat-field>
                  <button type="button" @click="() => onSubmit('test')" :loading="loading.campaigns" :disabled="isNew"
                    data-variant="primary" icon-left="email-outline">
                    {{ $t('campaigns.send') }}
                  </button>
                </oat-field>
              </div>
            </div>
          </div>
        </section>
      </oat-tab-item><!-- campaign -->

      <oat-tab-item :label="$t('campaigns.content')" icon="text" :disabled="isNew" value="content">
        <editor v-if="data.id" v-model="form.content" :id="data.id" :title="data.name" :disabled="!canEdit"
          :templates="templates" :content-types="contentTypes" />

        <div class="row">
          <div class="col-6">
            <p v-if="!isAttachFieldVisible" class="text-light text-7">
              <a href="#" @click.prevent="onShowAttachField()" data-cy="btn-attach">
                <oat-icon icon="file-upload-outline" />
                {{ $t('campaigns.addAttachments') }}
              </a>
            </p>

            <oat-field v-if="isAttachFieldVisible" :label="$t('campaigns.attachments')"
              data-cy="media">
              <oat-tag-input v-model="form.media" name="media" ref="media" field="filename"
                @focus="onOpenAttach" :disabled="!canEdit" />
            </oat-field>
          </div>
          <div class="col-12 align-right">
            <a href="https://listmonk.app/docs/templating/#template-expressions" target="_blank"
              rel="noopener noreferer">
              <oat-icon icon="code" /> {{ $t('campaigns.templatingRef') }}</a>
            <span v-if="canEdit && form.content.contentType !== 'plain'" class="text-light text-7 ml-6">
              <a v-if="form.altbody === null" href="#" @click.prevent="onAddAltBody">
                <oat-icon icon="text" /> {{ $t('campaigns.addAltText') }}
              </a>
              <a v-else href="#" @click.prevent="$utils.confirm(null, onRemoveAltBody)">
                <oat-icon icon="trash-can-outline" />
                {{ $t('campaigns.removeAltText') }}
              </a>
            </span>
          </div>
        </div>

        <div v-if="canEdit && form.content.contentType !== 'plain'" class="alt-body">
          <textarea aria-label="field" v-if="form.altbody !== null" v-model="form.altbody" :disabled="!canEdit" />
        </div>
      </oat-tab-item><!-- content -->

      <oat-tab-item :label="$t('globals.terms.attribs')" icon="code" value="attribs" :disabled="isNew">
        <section class="wrap">
          <oat-field :label="$t('globals.terms.attribs')" :message="$t('campaigns.attribsHelp')"
           >
            <textarea aria-label="field" v-model="form.attribsStr" :disabled="!canEdit" rows="15" />
          </oat-field>
        </section>
      </oat-tab-item><!-- attribs -->

      <oat-tab-item :label="$t('campaigns.archive')" icon="newspaper-variant-outline" value="archive" :disabled="isNew">
        <section class="wrap">
          <div class="row">
            <div class="col-4">
              <oat-field :label="$t('campaigns.archiveEnable')" data-cy="btn-archive"
                :message="$t('campaigns.archiveHelp')">
                <div class="row">
                  <div class="col-12">
                    <oat-switch data-cy="btn-archive" v-model="form.archive" :disabled="!canArchive" />
                  </div>
                  <div class="col-12">
                    <a :href="`${serverConfig.root_url}/archive/${data.uuid}`" target="_blank" rel="noopener noreferer"
                      :class="{ 'text-lighter': !form.archive }" aria-label="$t('campaigns.archive')">
                      <oat-icon icon="link-variant" />
                    </a>
                  </div>
                </div>
              </oat-field>
            </div>
            <div class="col-8">
              <oat-field>
                <oat-field v-if="!canEdit && canArchive">
                  <button type="button" @click="onUpdateCampaignArchive" :loading="loading.campaigns" data-variant="primary"
                    data-cy="btn-save">
                    {{ $t('globals.buttons.saveChanges') }}
                  </button>
                </oat-field>
              </oat-field>
            </div>
          </div>

          <div class="row">
            <div class="col-6">
              <oat-field :label="$tc('globals.terms.template')">
                <select aria-label="field" :placeholder="$tc('globals.terms.template')" v-model="form.archiveTemplateId" name="template"
                  :disabled="!canArchive || !form.archive || form.content.contentType === 'visual'" required>
                  <template v-for="t in templates">
                    <option v-if="t.type === 'campaign'" :value="t.id" :key="t.id">
                      {{ t.name }}
                    </option>
                  </template>
                </select>
              </oat-field>
            </div>

            <div class="col-6">
              <oat-field>
                <oat-field v-if="form.archive && (!this.form.archiveMetaStr || this.form.archiveMetaStr === '{}')">
                  <a class="button " href="#" @click.prevent="onFillArchiveMeta" aria-label="{}"><oat-icon
                      icon="code" /></a>
                </oat-field>
                <oat-field v-if="form.archive">
                  <button type="button" @click="onToggleArchivePreview" data-variant="primary"
                    data-cy="btn-preview">
                    {{ $t('campaigns.preview') }}
                  </button>
                </oat-field>
              </oat-field>
            </div>
          </div>
          <oat-field>
            <oat-field :label="$t('campaigns.archiveSlug')"
              :message="$t('campaigns.archiveSlugHelp')">
              <input aria-label="field" :maxlength="200" :ref="'focus'" v-model="form.archiveSlug" name="archive_slug"
                data-cy="archive-slug" :disabled="!canArchive || !form.archive">
            </oat-field>
          </oat-field>
          <oat-field :label="$t('campaigns.archiveMeta')" :message="$t('campaigns.archiveMetaHelp')"
           >
            <textarea aria-label="field" v-model="form.archiveMetaStr" name="archive_meta" data-cy="archive-meta"
              :disabled="!canArchive || !form.archive" rows="20" />
          </oat-field>
        </section>
      </oat-tab-item><!-- archive -->
    </oat-tabs>

    <oat-modal :active.sync="isAttachModalOpen" :width="900">
      <div class="dialog-card content" style="width: auto">
        <section class="dialog-body">
          <media is-modal @selected="onAttachSelect" />
        </section>
      </div>
    </oat-modal>

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
import CopyText from '../components/CopyText.vue';
import Editor from '../components/Editor.vue';
import ListSelector from '../components/ListSelector.vue';
import Media from './Media.vue';

export default Vue.extend({
  components: {
    ListSelector,
    Editor,
    Media,
    CopyText,
    CampaignPreview,
  },

  data() {
    return {
      contentTypes: Object.freeze({
        richtext: this.$t('campaigns.richText'),
        html: this.$t('campaigns.rawHTML'),
        markdown: this.$t('campaigns.markdown'),
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
      if (tab === 'content' && window.tinymce && window.tinymce.editors.length > 0) {
        this.$nextTick(() => {
          window.tinymce.editors[0].focus();
        });
      }

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
          this.$utils.toast(e.toString(), '');
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
          this.$utils.toast(e.toString(), '');
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
            '',

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

          this.$utils.toast(this.$t(typMsg, { name: d.name }));
          resolve();
        });
      });
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
  },

  computed: {
    ...mapState(['serverConfig', 'loading', 'lists', 'templates']),

    canManage() {
      return this.$can('campaigns:manage_all', 'campaigns:manage');
    },

    canSend() {
      return this.$can('campaigns:send');
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

    // eslint-disable-next-line func-names
    'data.sendAt': function () {
      if (this.data.sendAt !== null) {
        this.form.sendLater = true;
        this.form.sendAtDate = dayjs(this.data.sendAt).toDate();
      } else {
        this.form.sendLater = false;
        this.form.sendAtDate = null;
      }
    },
  },

  mounted() {
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

  beforeDestroy() {
    this.$events.$off('campaign.update');
  },
});
</script>
