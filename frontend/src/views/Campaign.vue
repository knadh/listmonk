<template>
  <section class="campaign">
    <header class="columns">
      <div class="column is-8">
        <p v-if="isEditing" class="tags">
          <b-tag v-if="isEditing" :class="data.status">{{ data.status }}</b-tag>
          <b-tag v-if="data.type === 'optin'" :class="data.type">{{ data.type }}</b-tag>
          <span v-if="isEditing" class="has-text-grey-light is-size-7">
            ID: {{ data.id }} / UUID: {{ data.uuid }}
          </span>
        </p>
        <h4 v-if="isEditing" class="title is-4">{{ data.name }}</h4>
        <h4 v-else class="title is-4">New campaign</h4>
      </div>

      <div class="column">
        <div class="buttons" v-if="isEditing && canEdit">
          <b-button @click="onSubmit" :loading="loading.campaigns"
            type="is-primary" icon-left="content-save-outline">Save changes</b-button>

          <b-button v-if="canStart" @click="startCampaign" :loading="loading.campaigns"
            type="is-primary" icon-left="rocket-launch-outline">
              Start campaign
          </b-button>
          <b-button v-if="canSchedule" @click="startCampaign" :loading="loading.campaigns"
            type="is-primary" icon-left="clock-start">
              Schedule campaign
          </b-button>
        </div>
      </div>
    </header>

    <b-loading :active="loading.campaigns"></b-loading>

    <b-tabs type="is-boxed" :animated="false" v-model="activeTab">
      <b-tab-item label="Campaign" icon="rocket-launch-outline">
        <section class="wrap">
          <div class="columns">
            <div class="column is-7">
              <form @submit.prevent="onSubmit">
                <b-field label="Name">
                  <b-input :maxlength="200" :ref="'focus'" v-model="form.name" :disabled="!canEdit"
                    placeholder="Name" required></b-input>
                </b-field>

                <b-field label="Subject">
                  <b-input :maxlength="200" v-model="form.subject" :disabled="!canEdit"
                    placeholder="Subject" required></b-input>
                </b-field>

                <b-field label="From address">
                  <b-input :maxlength="200" v-model="form.fromEmail" :disabled="!canEdit"
                    placeholder="Your Name <noreply@yoursite.com>" required></b-input>
                </b-field>

                <list-selector
                  v-model="form.lists"
                  :selected="form.lists"
                  :all="lists.results"
                  :disabled="!canEdit"
                  label="Lists"
                  placeholder="Lists to send to"
                ></list-selector>

                <b-field label="Template">
                  <b-select placeholder="Template" v-model="form.templateId"
                    :disabled="!canEdit" required>
                    <option v-for="t in templates" :value="t.id" :key="t.id">{{ t.name }}</option>
                  </b-select>
                </b-field>

                <b-field label="Tags">
                  <b-taginput v-model="form.tags" :disabled="!canEdit"
                    ellipsis icon="tag-outline" placeholder="Tags"></b-taginput>
                </b-field>
                <hr />

                <b-field label="Send later?">
                    <b-switch v-model="form.sendLater" :disabled="!canEdit"></b-switch>
                </b-field>

                <b-field v-if="form.sendLater" label="Send at">
                  <b-datetimepicker
                    v-model="form.sendAtDate"
                    :disabled="!canEdit"
                    placeholder="Date and time"
                    icon="calendar-clock"
                    :timepicker="{ hourFormat: '24' }"
                    :datetime-formatter="formatDateTime"
                    horizontal-time-picker>
                  </b-datetimepicker>
                </b-field>
                <hr />

                <b-field v-if="isNew">
                  <b-button native-type="submit" type="is-primary"
                    :loading="loading.campaigns">Continue</b-button>
                </b-field>
              </form>
            </div>
            <div class="column is-4 is-offset-1">
              <br />
              <div class="box">
                <h3 class="title is-size-6">Send test message</h3>
                  <b-field message="Hit Enter after typing an address to add multiple recipients.
                      The addresses must belong to existing subscribers.">
                    <b-taginput  v-model="form.testEmails"
                      :before-adding="$utils.validateEmail" :disabled="this.isNew"
                      ellipsis icon="email-outline" placeholder="E-mails"></b-taginput>
                  </b-field>
                  <b-field>
                    <b-button @click="sendTest" :loading="loading.campaigns" :disabled="this.isNew"
                      type="is-primary" icon-left="email-outline">Send</b-button>
                  </b-field>
              </div>
            </div>
          </div>
        </section>
      </b-tab-item><!-- campaign -->

      <b-tab-item label="Content" icon="text" :disabled="isNew">
        <section class="wrap">
          <editor
            v-model="form.content"
            :id="data.id"
            :title="data.name"
            :contentType="data.contentType"
            :body="data.body"
            :disabled="!canEdit"
          />
        </section>
      </b-tab-item><!-- content -->
    </b-tabs>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import dayjs from 'dayjs';
import ListSelector from '../components/ListSelector.vue';
import Editor from '../components/Editor.vue';

export default Vue.extend({
  components: {
    ListSelector,
    Editor,
  },

  data() {
    return {
      isNew: false,
      isEditing: false,
      activeTab: 0,

      data: {},

      // Binds form input values.
      form: {
        name: '',
        subject: '',
        fromEmail: window.CONFIG.fromEmail,
        templateId: 0,
        lists: [],
        tags: [],
        sendAt: null,
        content: { contentType: 'richtext', body: '' },

        // Parsed Date() version of send_at from the API.
        sendAtDate: null,
        sendLater: false,

        testEmails: [],
      },
    };
  },

  methods: {
    formatDateTime(s) {
      return dayjs(s).format('YYYY-MM-DD HH:mm');
    },

    getCampaign(id) {
      return this.$api.getCampaign(id).then((r) => {
        this.data = r.data;
        this.form = { ...this.form, ...r.data };

        if (r.data.sendAt !== null) {
          this.form.sendLater = true;
          this.form.sendAtDate = dayjs(r.data.sendAt).toDate();
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
        content_type: 'richtext',
        messenger: 'email',
        type: 'regular',
        tags: this.form.tags,
        template_id: this.form.templateId,
        body: this.form.body,
        subscribers: this.form.testEmails,
      };

      this.$api.testCampaign(data).then(() => {
        this.$utils.toast('Test message sent');
      });
      return false;
    },

    onSubmit() {
      if (this.isNew) {
        this.createCampaign();
      } else {
        this.updateCampaign();
      }
    },

    createCampaign() {
      const data = {
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.fromEmail,
        content_type: 'richtext',
        messenger: 'email',
        type: 'regular',
        tags: this.form.tags,
        template_id: this.form.templateId,
        // body: this.form.body,
      };

      this.$api.createCampaign(data).then((r) => {
        this.$router.push({ name: 'campaign', hash: '#content', params: { id: r.data.id } });

        // this.data = r.data;
        // this.isEditing = true;
        // this.isNew = false;
        // this.activeTab = 1;
      });
      return false;
    },

    async updateCampaign(typ) {
      const data = {
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        from_email: this.form.fromEmail,
        messenger: 'email',
        type: 'regular',
        tags: this.form.tags,
        send_later: this.form.sendLater,
        send_at: this.form.sendLater ? this.form.sendAtDate : null,
        template_id: this.form.templateId,
        content_type: this.form.content.contentType,
        body: this.form.content.body,
      };

      let typMsg = 'updated';
      if (typ === 'start') {
        typMsg = 'started';
      }

      // This promise is used by startCampaign to first save before starting.
      return new Promise((resolve) => {
        this.$api.updateCampaign(this.data.id, data).then((resp) => {
          this.data = resp.data;
          this.$buefy.toast.open({
            message: `'${resp.data.name}' ${typMsg}`,
            type: 'is-success',
            queue: false,
          });
          resolve();
        });
      });
    },

    // Starts or schedule a campaign.
    startCampaign() {
      let status = '';
      if (this.canStart) {
        status = 'running';
      } else if (this.canSchedule) {
        status = 'scheduled';
      } else {
        return;
      }

      this.$utils.confirm(null,
        () => {
          // First save the campaign.
          this.updateCampaign().then(() => {
            // Then start/schedule it.
            this.$api.changeCampaignStatus(this.data.id, status).then(() => {
              this.$router.push({ name: 'campaigns' });
            });
          });
        });
    },
  },

  computed: {
    ...mapState(['lists', 'templates', 'loading']),

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
  },

  mounted() {
    const { id } = this.$route.params;

    // New campaign.
    if (id === 'new') {
      this.isNew = true;
    } else {
      const intID = parseInt(id, 10);
      if (intID <= 0 || Number.isNaN(intID)) {
        this.$buefy.toast.open({
          message: 'Invalid campaign',
          type: 'is-danger',
          queue: false,
        });
        return;
      }

      this.isEditing = true;
    }

    // Get templates list.
    this.$api.getTemplates().then((r) => {
      if (r.data.length > 0) {
        if (!this.form.templateId) {
          this.form.templateId = r.data.find((i) => i.isDefault === true).id;
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
    }

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
