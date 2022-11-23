<template>
  <section class="campaign">
    <header class="columns page-header">
      <div class="column is-6">
        <p v-if="isEditing && data.status" class="tags">
          <b-tag v-if="isEditing" :class="data.status">
            {{ $t(`campaigns.status.${data.status}`) }}
          </b-tag>
          <b-tag v-if="data.type === 'optin'" :class="data.type">
            {{ $t('lists.optin') }}
          </b-tag>
          <span v-if="isEditing" class="has-text-grey-light is-size-7" :data-campaign-id="data.id">
            {{ $t('globals.fields.id') }}: {{ data.id }} /
            {{ $t('globals.fields.uuid') }}: {{ data.uuid }}
          </span>
        </p>
        <h4 v-if="isEditing" class="title is-4">{{ data.name }}</h4>
        <h4 v-else class="title is-4">{{ $t('campaigns.newCampaign') }}</h4>
      </div>

      <div class="column is-6">
        <div class="buttons">
          <b-field grouped v-if="isEditing && canEdit">
            <b-field expanded>
              <b-button  expanded @click="() => onSubmit('update')" :loading="loading.campaigns"
                type="is-primary" icon-left="content-save-outline" data-cy="btn-save">
                {{ $t('globals.buttons.saveChanges') }}
              </b-button>
            </b-field>
            <b-field expanded v-if="canStart">
              <b-button  expanded @click="startCampaign" :loading="loading.campaigns"
                type="is-primary" icon-left="rocket-launch-outline" data-cy="btn-start">
                {{ $t('campaigns.start') }}
              </b-button>
            </b-field>
            <b-field expanded v-if="canSchedule">
              <b-button  expanded @click="startCampaign"
                :loading="loading.campaigns"
                type="is-primary" icon-left="clock-start" data-cy="btn-schedule">
                {{ $t('campaigns.schedule') }}
              </b-button>
            </b-field>
          </b-field>
        </div>
      </div>
    </header>

    <b-loading :active="loading.campaigns"></b-loading>

    <b-tabs type="is-boxed" :animated="false" v-model="activeTab" @input="onTab">
      <b-tab-item :label="$tc('globals.terms.campaign')" label-position="on-border"
        value="campaign" icon="rocket-launch-outline">
        <section class="wrap">
          <div class="columns">
            <div class="column is-7">
              <form @submit.prevent="() => onSubmit(isNew ? 'create' : 'update')">
                <b-field :label="$t('globals.fields.name')" label-position="on-border">
                  <b-input :maxlength="200" :ref="'focus'" v-model="form.name"
                    name="name" :disabled="!canEdit"
                    :placeholder="$t('globals.fields.name')" required></b-input>
                </b-field>

                <b-field :label="$t('campaigns.subject')" label-position="on-border">
                  <b-input :maxlength="200" v-model="form.subject"
                    name="subject" :disabled="!canEdit"
                    :placeholder="$t('campaigns.subject')" required></b-input>
                </b-field>

                <b-field :label="$t('campaigns.fromAddress')" label-position="on-border">
                  <b-input :maxlength="200" v-model="form.fromEmail"
                    name="from_email" :disabled="!canEdit"
                    :placeholder="$t('campaigns.fromAddressPlaceholder')" required></b-input>
                </b-field>

                <list-selector
                  v-model="form.lists"
                  :selected="form.lists"
                  :all="lists.results"
                  :disabled="!canEdit"
                  :label="$t('globals.terms.lists')"
                  :placeholder="$t('campaigns.sendToLists')"
                ></list-selector>

                <b-field :label="$tc('globals.terms.template')" label-position="on-border">
                  <b-select :placeholder="$tc('globals.terms.template')" v-model="form.templateId"
                    name="template" :disabled="!canEdit" required>
                    <template v-for="t in templates">
                      <option v-if="t.type === 'campaign'"
                        :value="t.id" :key="t.id">{{ t.name }}</option>
                    </template>
                  </b-select>
                </b-field>

                <b-field :label="$tc('globals.terms.messenger')" label-position="on-border">
                  <b-select :placeholder="$tc('globals.terms.messenger')" v-model="form.messenger"
                    name="messenger" :disabled="!canEdit" required>
                    <option v-for="m in messengers"
                      :value="m" :key="m">{{ m }}</option>
                  </b-select>
                </b-field>

                <b-field :label="$t('globals.terms.tags')" label-position="on-border">
                  <b-taginput v-model="form.tags" name="tags" :disabled="!canEdit"
                    ellipsis icon="tag-outline" :placeholder="$t('globals.terms.tags')" />
                </b-field>
                <hr />

                <div class="columns">
                  <div class="column is-4">
                    <b-field :label="$t('campaigns.sendLater')" data-cy="btn-send-later">
                        <b-switch v-model="form.sendLater" :disabled="!canEdit" />
                    </b-field>
                  </div>
                  <div class="column">
                    <br />
                    <b-field v-if="form.sendLater" data-cy="send_at"
                      :message="form.sendAtDate ? $utils.duration(Date(), form.sendAtDate) : ''">
                      <b-datetimepicker
                        v-model="form.sendAtDate"
                        :disabled="!canEdit"
                        :placeholder="$t('campaigns.dateAndTime')"
                        icon="calendar-clock"
                        :timepicker="{ hourFormat: '24' }"
                        :datetime-formatter="formatDateTime"
                        horizontal-time-picker>
                      </b-datetimepicker>
                    </b-field>
                  </div>
                </div>

                <div>
                  <p class="has-text-right">
                    <a href="#" @click.prevent="showHeaders" data-cy="btn-headers">
                      <b-icon icon="plus" />{{ $t('settings.smtp.setCustomHeaders') }}
                    </a>
                  </p>
                  <b-field v-if="form.headersStr !== '[]' || isHeadersVisible"
                    label-position="on-border" :message="$t('campaigns.customHeadersHelp')">
                    <b-input v-model="form.headersStr" name="headers" type="textarea"
                      placeholder='[{"X-Custom": "value"}, {"X-Custom2": "value"}]'
                      :disabled="!canEdit" />
                  </b-field>
                </div>
                <hr />

                <b-field v-if="isNew">
                  <b-button native-type="submit" type="is-primary"
                    :loading="loading.campaigns" data-cy="btn-continue">
                    {{ $t('campaigns.continue') }}
                  </b-button>
                </b-field>
              </form>
            </div>
            <div class="column is-4 is-offset-1">
              <br />
              <div class="box">
                <h3 class="title is-size-6">{{ $t('campaigns.sendTest') }}</h3>
                  <b-field :message="$t('campaigns.sendTestHelp')">
                    <b-taginput v-model="form.testEmails"
                      :before-adding="$utils.validateEmail" :disabled="isNew"
                      ellipsis icon="email-outline" :placeholder="$t('campaigns.testEmails')" />
                  </b-field>
                  <b-field>
                    <b-button @click="() => onSubmit('test')" :loading="loading.campaigns"
                      :disabled="isNew" type="is-primary" icon-left="email-outline">
                      {{ $t('campaigns.send') }}
                    </b-button>
                  </b-field>
              </div>
            </div>
          </div>
        </section>
      </b-tab-item><!-- campaign -->

      <b-tab-item :label="$t('campaigns.content')" icon="text" :disabled="isNew" value="content">
        <editor
          v-model="form.content"
          :id="data.id"
          :title="data.name"
          :templateId="form.templateId"
          :contentType="data.contentType"
          :body="data.body"
          :disabled="!canEdit"
        />

        <div v-if="canEdit && form.content.contentType !== 'plain'" class="alt-body">
          <p class="is-size-6 has-text-grey has-text-right">
            <a v-if="form.altbody === null" href="#" @click.prevent="addAltBody">
              <b-icon icon="text" size="is-small" /> {{ $t('campaigns.addAltText') }}
            </a>
            <a v-else href="#" @click.prevent="$utils.confirm(null, removeAltBody)">
              <b-icon icon="trash-can-outline" size="is-small" />
              {{ $t('campaigns.removeAltText') }}
            </a>
          </p>
          <br />
          <b-input v-if="form.altbody !== null" v-model="form.altbody"
            type="textarea" :disabled="!canEdit" />
        </div>
      </b-tab-item><!-- content -->

      <b-tab-item :label="$t('campaigns.archive')" icon="newspaper-variant-outline"
        value="archive" :disabled="isNew">
        <section class="wrap">
          <b-field :label="$t('campaigns.archiveEnable')" data-cy="btn-archive"
            :message="$t('campaigns.archiveHelp')">
            <div class="columns">
              <div class="column">
                <b-switch data-cy="btn-archive" v-model="form.archive" :disabled="!canArchive" />
              </div>
              <div class="column is-12">
                <a :href="`${settings['app.root_url']}/archive/${data.uuid}`" target="_blank"
                  :class="{'has-text-grey-light': !form.archive}">
                  <b-icon icon="link-variant" />
                </a>
              </div>
            </div>
          </b-field>

          <div class="columns">
            <div class="column is-8">
              <b-field :label="$tc('globals.terms.template')" label-position="on-border">
                <b-select :placeholder="$tc('globals.terms.template')"
                  v-model="form.archiveTemplateId" name="template"
                  :disabled="!canArchive || !form.archive" required>
                  <template v-for="t in templates">
                    <option v-if="t.type === 'campaign'"
                      :value="t.id" :key="t.id">{{ t.name }}</option>
                  </template>
                </b-select>
              </b-field>
            </div>

            <div class="column has-text-right">
              <a v-if="!this.form.archiveMetaStr || this.form.archiveMetaStr === '{}'"
                class="button" href="#" @click.prevent="onFillArchiveMeta">{}</a>
            </div>
          </div>
          <b-field :label="$t('campaigns.archiveMeta')"
            :message="$t('campaigns.archiveMetaHelp')" label-position="on-border">
            <b-input v-model="form.archiveMetaStr" name="archive_meta" type="textarea"
              data-cy="archive-meta" :disabled="!canArchive || !form.archive" rows="20" />
          </b-field>

          <b-field v-if="!canEdit && canArchive">
            <b-button @click="onUpdateCampaignArchive" :loading="loading.campaigns"
              type="is-primary" icon-left="content-save-outline" data-cy="btn-archive-save">
              {{ $t('globals.buttons.saveChanges') }}
            </b-button>
          </b-field>
        </section>
      </b-tab-item><!-- archive -->
    </b-tabs>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import dayjs from 'dayjs';
import htmlToPlainText from 'textversionjs';

import ListSelector from '../components/ListSelector.vue';
import Editor from '../components/Editor.vue';

const TABS = ['campaign', 'content', 'archive'];

export default Vue.extend({
  components: {
    ListSelector,
    Editor,
  },

  data() {
    return {
      isNew: false,
      isEditing: false,
      isHeadersVisible: false,
      activeTab: 0,

      data: {},

      // IDs from ?list_id query param.
      selListIDs: [],

      // Binds form input values.
      form: {
        name: '',
        subject: '',
        fromEmail: '',
        headersStr: '[]',
        headers: [],
        messenger: 'email',
        templateId: 0,
        lists: [],
        tags: [],
        sendAt: null,
        content: { contentType: 'richtext', body: '' },
        altbody: null,

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

    addAltBody() {
      this.form.altbody = htmlToPlainText(this.form.content.body);
    },

    removeAltBody() {
      this.form.altbody = null;
    },

    showHeaders() {
      this.isHeadersVisible = !this.isHeadersVisible;
    },

    isUnsaved() {
      return this.data.body !== this.form.content.body
        || this.data.contentType !== this.form.content.contentType;
    },

    onTab(t) {
      const tab = TABS[t];
      if (tab === 'content' && window.tinymce && window.tinymce.editors.length > 0) {
        this.$nextTick(() => {
          window.tinymce.editors[0].focus();
        });
      }
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
      } else {
        this.form.archiveMeta = {};
      }

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

          // The structure that is populated by editor input event.
          content: { contentType: data.contentType, body: data.body },
        };

        if (data.sendAt !== null) {
          this.form.sendLater = true;
          this.form.sendAtDate = dayjs(data.sendAt).toDate();
        }
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
        template_id: this.form.templateId,
        content_type: this.form.content.contentType,
        body: this.form.content.body,
        altbody: this.form.content.contentType !== 'plain' ? this.form.altbody : null,
        subscribers: this.form.testEmails,
      };

      this.$api.testCampaign(data).then(() => {
        this.$utils.toast(this.$t('campaigns.testSent'));
      });
      return false;
    },

    createCampaign() {
      const data = {
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.fromEmail,
        content_type: 'richtext',
        messenger: this.form.messenger,
        type: 'regular',
        tags: this.form.tags,
        send_later: this.form.sendLater,
        send_at: this.form.sendLater ? this.form.sendAtDate : null,
        headers: this.form.headers,
        template_id: this.form.templateId,
        // body: this.form.body,
      };

      this.$api.createCampaign(data).then((d) => {
        this.$router.push({ name: 'campaign', hash: '#content', params: { id: d.id } });
      });
      return false;
    },

    async updateCampaign(typ) {
      const data = {
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.fromEmail,
        messenger: this.form.messenger,
        type: 'regular',
        tags: this.form.tags,
        send_later: this.form.sendLater,
        send_at: this.form.sendLater ? this.form.sendAtDate : null,
        headers: this.form.headers,
        template_id: this.form.templateId,
        content_type: this.form.content.contentType,
        body: this.form.content.body,
        altbody: this.form.content.contentType !== 'plain' ? this.form.altbody : null,
        archive: this.form.archive,
        archive_template_id: this.form.archiveTemplateId,
        archive_meta: this.form.archiveMeta,
      };

      let typMsg = 'globals.messages.updated';
      if (typ === 'start') {
        typMsg = 'campaigns.started';
      }

      // This promise is used by startCampaign to first save before starting.
      return new Promise((resolve) => {
        this.$api.updateCampaign(this.data.id, data).then((d) => {
          this.data = d;
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
      };

      this.$api.updateCampaignArchive(this.data.id, data);
    },

    // Starts or schedule a campaign.
    startCampaign() {
      if (!this.canStart && !this.canSchedule) {
        return;
      }

      this.$utils.confirm(null,
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
        });
    },
  },

  computed: {
    ...mapState(['settings', 'loading', 'lists', 'templates']),

    canEdit() {
      return this.isNew
        || this.data.status === 'draft' || this.data.status === 'scheduled';
    },

    canSchedule() {
      return this.data.status === 'draft' && this.data.sendAt;
    },

    canStart() {
      return this.data.status === 'draft' && !this.data.sendAt;
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

    messengers() {
      return ['email', ...this.settings.messengers.map((m) => m.name)];
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
  },

  mounted() {
    window.onbeforeunload = () => this.isUnsaved() || null;

    // Fill default form fields.
    this.form.fromEmail = this.settings['app.from_email'];

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
          this.form.templateId = data.find((i) => i.isDefault === true).id;
        }
      }
    });

    // Fetch campaign.
    if (this.isEditing) {
      this.getCampaign(id).then(() => {
        if (this.$route.hash === '#content') {
          this.activeTab = 1;
        }
      });
    } else {
      this.form.messenger = 'email';
    }

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
