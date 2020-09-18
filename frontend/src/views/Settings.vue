<template>
  <section class="settings">
    <b-loading :is-full-page="true" v-if="isLoading" active />
    <header class="columns">
      <div class="column is-half">
        <h1 class="title is-4">Settings</h1>
      </div>
      <div class="column has-text-right">
        <b-button :disabled="!hasFormChanged"
          type="is-primary" icon-left="content-save-outline"
          @click="onSubmit" class="isSaveEnabled">Save changes</b-button>
      </div>
    </header>
    <hr />

    <section class="wrap-small">
      <form @submit.prevent="onSubmit">
        <b-tabs type="is-boxed" :animated="false">
          <b-tab-item label="General" label-position="on-border">
            <div class="items">
              <b-field label="Root URL" label-position="on-border"
                message="Public URL of the installation (no trailing slash).">
                <b-input v-model="form['app.root_url']" name="app.root_url"
                    placeholder='https://listmonk.yoursite.com' :maxlength="300" />
              </b-field>

              <b-field label="Logo URL" label-position="on-border"
                message="(Optional) full URL to the static logo to be displayed on
                        user facing view such as the unsubscription page.">
                <b-input v-model="form['app.logo_url']" name="app.logo_url"
                    placeholder='https://listmonk.yoursite.com/logo.png' :maxlength="300" />
              </b-field>

              <b-field label="Favicon URL" label-position="on-border"
                message="(Optional) full URL to the static favicon to be displayed on
                        user facing view such as the unsubscription page.">
                <b-input v-model="form['app.favicon_url']" name="app.favicon_url"
                    placeholder='https://listmonk.yoursite.com/favicon.png' :maxlength="300" />
              </b-field>

              <hr />
              <b-field label="Default 'from' email" label-position="on-border"
                message="(Optional) full URL to the static logo to be displayed on
                        user facing view such as the unsubscription page.">
                <b-input v-model="form['app.from_email']" name="app.from_email"
                    placeholder='Listmonk <noreply@listmonk.yoursite.com>'
                    pattern="(.+?)\s<(.+?)@(.+?)>" :maxlength="300" />
              </b-field>

              <b-field label="Admin notification e-mails" label-position="on-border"
                message="Comma separated list of e-mail addresses to which admin
                        notifications such as import updates, campaign completion,
                        failure etc. should be sent.">
                <b-taginput v-model="form['app.notify_emails']" name="app.notify_emails"
                  :before-adding="(v) => v.match(/(.+?)@(.+?)/)"
                  placeholder='you@yoursite.com' />
              </b-field>
            </div>
          </b-tab-item><!-- general -->

          <b-tab-item label="Performance">
            <div class="items">
              <b-field label="Concurrency" label-position="on-border"
                message="Maximum concurrent worker (threads) that will attempt to send messages
                        simultaneously.">
                <b-numberinput v-model="form['app.concurrency']"
                    name="app.concurrency" type="is-light"
                    placeholder="5" min="1" max="10000" />
              </b-field>

              <b-field label="Message rate" label-position="on-border"
                message="Maximum number of messages to be sent out per second
                        per worker in a second. If concurrency = 10 and message_rate = 10,
                        then up to 10x10=100 messages may be pushed out every second.
                        This, along with concurrency, should be tweaked to keep the
                        net messages going out per second under the target
                        message servers rate limits if any.">
                <b-numberinput v-model="form['app.message_rate']"
                    name="app.message_rate" type="is-light"
                    placeholder="5" min="1" max="100000" />
              </b-field>

              <b-field label="Batch size" label-position="on-border"
                message="The number of subscribers to pull from the databse in a single iteration.
                        Each iteration pulls subscribers from the database, sends messages to them,
                        and then moves on to the next iteration to pull the next batch.
                        This should ideally be higher than the maximum achievable
                        throughput (concurrency * message_rate).">
                <b-numberinput v-model="form['app.batch_size']"
                    name="app.batch_size" type="is-light"
                    placeholder="1000" min="1" max="100000" />
              </b-field>

              <b-field label="Maximum error threshold" label-position="on-border"
                message="The number of errors (eg: SMTP timeouts while e-mailing) a running
                        campaign should tolerate before it is paused for manual
                        investigation or intervention. Set to 0 to never pause.">
                <b-numberinput v-model="form['app.max_send_errors']"
                    name="app.max_send_errors" type="is-light"
                    placeholder="1999" min="0" max="100000" />
              </b-field>
            </div>
          </b-tab-item><!-- performance -->

          <b-tab-item label="Privacy">
            <div class="items">
              <b-field label="Include `List-Unsubscribe` header"
                message="Include unsubscription headers that allow e-mail clients to
                        allow users to unsubscribe in a single click.">
                <b-switch v-model="form['privacy.unsubscribe_header']"
                    name="privacy.unsubscribe_header" />
              </b-field>

              <b-field label="Allow blocklisting"
                message="Allow subscribers to unsubscribe from all mailing lists and mark
                      themselves as blocklisted?">
                <b-switch v-model="form['privacy.allow_blocklist']"
                    name="privacy.allow_blocklist" />
              </b-field>

              <b-field label="Allow exporting"
                message="Allow subscribers to export data collected on them?">
                <b-switch v-model="form['privacy.allow_export']"
                    name="privacy.allow_export" />
              </b-field>

              <b-field label="Allow wiping"
                message="Allow subscribers to delete themselves including their
                      subscriptions and all other data from the database.
                      Campaign views and link clicks are also
                      removed while views and click counts remain (with no subscriber
                      associated to them) so that stats and analytics aren't affected.">
                <b-switch v-model="form['privacy.allow_wipe']"
                    name="privacy.allow_wipe" />
              </b-field>
            </div>
          </b-tab-item><!-- privacy -->

          <b-tab-item label="Media uploads">
            <div class="items">
              <b-field label="Provider" label-position="on-border">
                <b-select v-model="form['upload.provider']" name="upload.provider">
                  <option value="filesystem">filesystem</option>
                  <option value="s3">s3</option>
                </b-select>
              </b-field>

              <div class="block" v-if="form['upload.provider'] === 'filesystem'">
                <b-field label="Upload path" label-position="on-border"
                  message="Path to the directory where media will be uploaded.">
                  <b-input v-model="form['upload.filesystem.upload_path']"
                      name="app.upload_path" placeholder='/home/listmonk/uploads'
                      :maxlength="200" />
                </b-field>

                <b-field label="Upload URI" label-position="on-border"
                  message="Upload URI that's visible to the outside world.
                        The media uploaded to upload_path will be publicly accessible
                        under {root_url}/{}, for instance, https://listmonk.yoursite.com/uploads.">
                  <b-input v-model="form['upload.filesystem.upload_uri']"
                      name="app.upload_uri" placeholder='/uploads' :maxlength="200" />
                </b-field>
              </div><!-- filesystem -->

              <div class="block" v-if="form['upload.provider'] === 's3'">
                <div class="columns">
                  <div class="column is-3">
                    <b-field label="Region" label-position="on-border" expanded>
                      <b-input v-model="form['upload.s3.aws_default_region']"
                          name="upload.s3.aws_default_region"
                          :maxlength="200" placeholder="ap-south-1" />
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field grouped>
                      <b-field label="AWS access key" label-position="on-border" expanded>
                        <b-input v-model="form['upload.s3.aws_access_key_id']"
                            name="upload.s3.aws_access_key_id" :maxlength="200" />
                      </b-field>
                      <b-field label="AWS access secret" label-position="on-border" expanded
                        message="Enter a value to change.">
                        <b-input v-model="form['upload.s3.aws_secret_access_key']"
                            name="upload.s3.aws_secret_access_key" type="password"
                            :maxlength="200" />
                      </b-field>
                    </b-field>
                  </div>
                </div>

                <div class="columns">
                  <div class="column is-3">
                    <b-field label="Bucket type" label-position="on-border">
                      <b-select v-model="form['upload.s3.bucket_type']"
                        name="upload.s3.bucket_type" expanded>
                        <option value="private">private</option>
                        <option value="public">public</option>
                      </b-select>
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field grouped>
                      <b-field label="Bucket" label-position="on-border" expanded>
                        <b-input v-model="form['upload.s3.bucket']"
                            name="upload.s3.bucket" :maxlength="200" placeholder="" />
                      </b-field>
                      <b-field label="Bucket path" label-position="on-border"
                        message="Path inside the bucket to upload files. Default is /" expanded>
                        <b-input v-model="form['upload.s3.bucket_path']"
                            name="upload.s3.bucket_path" :maxlength="200" placeholder="/" />
                      </b-field>
                    </b-field>
                  </div>
                </div>
                <div class="columns">
                  <div class="column is-3">
                    <b-field label="Upload expiry" label-position="on-border"
                      message="(Optional) Specify TTL (in seconds) for the generated presigned URL.
                              Only applicable for private buckets
                              (s, m, h, d for seconds, minutes, hours, days)." expanded>
                      <b-input v-model="form['upload.s3.expiry']"
                        name="upload.s3.expiry"
                        placeholder="14d" :pattern="regDuration" :maxlength="10" />
                    </b-field>
                  </div>
                </div>
              </div><!-- s3 -->
            </div>
          </b-tab-item><!-- media -->

          <b-tab-item label="SMTP">
            <div class="items mail-servers">
              <div class="block box" v-for="(item, n) in form.smtp" :key="n">
                <div class="columns">
                  <div class="column is-2">
                    <b-field label="Enabled">
                      <b-switch v-model="item.enabled" name="enabled"
                          :native-value="true" />
                    </b-field>
                    <b-field v-if="form.smtp.length > 1">
                      <a @click.prevent="$utils.confirm(null, () => removeSMTP(n))"
                        href="#" class="is-size-7">
                        <b-icon icon="trash-can-outline" size="is-small" /> Delete
                      </a>
                    </b-field>
                  </div><!-- first column -->

                  <div class="column" :class="{'disabled': !item.enabled}">
                    <div class="columns">
                      <div class="column is-8">
                        <b-field label="Host" label-position="on-border"
                          message="SMTP server's host address.">
                          <b-input v-model="item.host" name="host"
                            placeholder='smtp.yourmailserver.net' :maxlength="200" />
                        </b-field>
                      </div>
                      <div class="column">
                        <b-field label="Port" label-position="on-border"
                          message="SMTP server's port.">
                          <b-numberinput v-model="item.port" name="port" type="is-light"
                              controls-position="compact"
                              placeholder="25" min="1" max="65535" />
                        </b-field>
                      </div>
                    </div><!-- host -->

                    <div class="columns">
                      <div class="column is-2">
                        <b-field label="Auth protocol" label-position="on-border">
                          <b-select v-model="item.auth_protocol" name="auth_protocol">
                            <option value="none">none</option>
                            <option value="cram">cram</option>
                            <option value="plain">plain</option>
                            <option value="login">login</option>
                          </b-select>
                        </b-field>
                      </div>
                      <div class="column">
                        <b-field grouped>
                          <b-field label="Username" label-position="on-border" expanded>
                            <b-input v-model="item.username"
                              :disabled="item.auth_protocol === 'none'"
                              name="username" placeholder="mysmtp" :maxlength="200" />
                          </b-field>
                          <b-field label="Password" label-position="on-border" expanded
                            message="Enter a value to change.">
                            <b-input v-model="item.password"
                              :disabled="item.auth_protocol === 'none'"
                              name="password" type="password" placeholder="Enter to change"
                              :maxlength="200" />
                          </b-field>
                        </b-field>
                      </div>
                    </div><!-- auth -->
                    <hr />

                    <div class="columns">
                      <div class="column is-6">
                        <b-field label="HELO hostname" label-position="on-border"
                          message="Optional. Some SMTP servers require a FQDN in the hostname.
                                By default, HELLOs go with 'localhost'. Set this if a custom
                                hostname should be used.">
                          <b-input v-model="item.hello_hostname"
                            name="hello_hostname" placeholder="" :maxlength="200" />
                        </b-field>
                      </div>
                      <div class="column">
                        <b-field grouped>
                          <b-field label="TLS" expanded
                            message="Enable STARTTLS.">
                            <b-switch v-model="item.tls_enabled" name="item.tls_enabled" />
                          </b-field>
                          <b-field label="Skip TLS verification" expanded
                            message="Skip hostname check on the TLS certificate.">
                            <b-switch v-model="item.tls_skip_verify"
                              :disabled="!item.tls_enabled" name="item.tls_skip_verify" />
                          </b-field>
                        </b-field>
                      </div>
                    </div><!-- TLS -->
                    <hr />

                    <div class="columns">
                      <div class="column is-3">
                        <b-field label="Max. connections" label-position="on-border"
                          message="Maximum concurrent connections to the SMTP server.">
                          <b-numberinput v-model="item.max_conns" name="max_conns" type="is-light"
                              controls-position="compact"
                              placeholder="25" min="1" max="65535" />
                        </b-field>
                      </div>
                      <div class="column is-3">
                        <b-field label="Retries" label-position="on-border"
                          message="The number of times a message should be retried
                                  if sending fails.">
                          <b-numberinput v-model="item.max_msg_retries" name="max_msg_retries"
                              type="is-light"
                              controls-position="compact"
                              placeholder="2" min="1" max="1000" />
                        </b-field>
                      </div>
                      <div class="column is-3">
                        <b-field label="Idle timeout" label-position="on-border"
                          message="Time to wait for new activity on a connection before closing
                                  it and removing it from the pool (s for second, m for minute).">
                          <b-input v-model="item.idle_timeout" name="idle_timeout"
                            placeholder="15s" :pattern="regDuration" :maxlength="10" />
                        </b-field>
                      </div>
                      <div class="column is-3">
                        <b-field label="Wait timeout" label-position="on-border"
                          message="Time to wait for new activity on a connection before closing
                                  it and removing it from the pool (s for second, m for minute).">
                          <b-input v-model="item.wait_timeout" name="wait_timeout"
                            placeholder="5s" :pattern="regDuration" :maxlength="10" />
                        </b-field>
                      </div>
                    </div>
                    <hr />

                    <div>
                      <p v-if="item.email_headers.length === 0 && !item.showHeaders">
                        <a href="#" class="is-size-7" @click.prevent="() => showSMTPHeaders(n)">
                          <b-icon icon="plus" />Set custom headers</a>
                      </p>
                      <b-field v-if="item.email_headers.length > 0 || item.showHeaders"
                        label="Custom headers" label-position="on-border"
                        message='Optional array of e-mail headers to include in all messages
                          sent from this server.
                          eg: [{"X-Custom": "value"}, {"X-Custom2": "value"}]'>
                        <b-input v-model="item.strEmailHeaders" name="email_headers" type="textarea"
                          placeholder='[{"X-Custom": "value"}, {"X-Custom2": "value"}]' />
                      </b-field>
                    </div>
                  </div>
                </div><!-- second container column -->
              </div><!-- block -->
            </div><!-- mail-servers -->

            <b-button @click="addSMTP" icon-left="plus" type="is-primary">Add new</b-button>
          </b-tab-item><!-- mail servers -->
        </b-tabs>
      </form>
    </section>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import store from '../store';
import { models } from '../constants';

const dummyPassword = ' '.repeat(8);

export default Vue.extend({
  data() {
    return {
      regDuration: '[0-9]+(ms|s|m|h|d)',
      isLoading: true,

      // formCopy is a stringified copy of the original settings against which
      // form is compared to detect changes.
      formCopy: '',
      form: {},
    };
  },

  methods: {
    addSMTP() {
      const [data] = JSON.parse(JSON.stringify(this.form.smtp.slice(-1)));
      this.form.smtp.push(data);
    },

    removeSMTP(i) {
      this.form.smtp.splice(i, 1);
    },

    showSMTPHeaders(i) {
      const s = this.form.smtp[i];
      s.showHeaders = true;
      this.form.smtp.splice(i, 1, s);
    },

    onSubmit() {
      const form = JSON.parse(JSON.stringify(this.form));

      // De-serialize custom e-mail headers.
      for (let i = 0; i < form.smtp.length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form.smtp[i].password === dummyPassword) {
          form.smtp[i].password = '';
        }

        if (form.smtp[i].strEmailHeaders && form.smtp[i].strEmailHeaders !== '[]') {
          form.smtp[i].email_headers = JSON.parse(form.smtp[i].strEmailHeaders);
        } else {
          form.smtp[i].email_headers = [];
        }
      }

      if (form['upload.s3.aws_secret_access_key'] === dummyPassword) {
        form['upload.s3.aws_secret_access_key'] = '';
      }

      this.isLoading = true;
      this.$api.updateSettings(form).then((data) => {
        if (data.needsRestart) {
          // Update the 'needsRestart' flag on the global serverConfig state
          // as there are running campaigns and the app couldn't auto-restart.
          store.commit('setModelResponse',
            { model: models.serverConfig, data: { ...this.serverConfig, needsRestart: true } });
          this.getSettings();
          return;
        }

        this.$utils.toast('Settings saved. Reloading app ...');

        // Poll until there's a 200 response, waiting for the app
        // to restart and come back up.
        const pollId = setInterval(() => {
          this.$api.getHealth().then(() => {
            clearInterval(pollId);
            this.getSettings();
          });
        }, 500);
      }, () => {
        this.isLoading = false;
      });
    },

    getSettings() {
      this.$api.getSettings().then((data) => {
        const d = data;
        // Serialize the `email_headers` array map to display on the form.
        for (let i = 0; i < d.smtp.length; i += 1) {
          d.smtp[i].strEmailHeaders = JSON.stringify(d.smtp[i].email_headers, null, 4);

          // The backend doesn't send passwords, so add a dummy so that it
          // the password looks filled on the UI.
          d.smtp[i].password = dummyPassword;
        }

        if (d['upload.provider'] === 's3') {
          d['upload.s3.aws_secret_access_key'] = dummyPassword;
        }

        this.form = d;
        this.formCopy = JSON.stringify(d);
        this.isLoading = false;
      });
    },
  },

  computed: {
    ...mapState(['serverConfig', 'loading']),

    hasFormChanged() {
      if (!this.formCopy) {
        return false;
      }
      return JSON.stringify(this.form) !== this.formCopy;
    },
  },

  beforeRouteLeave(to, from, next) {
    if (this.hasFormChanged) {
      this.$utils.confirm('Discard changes?', () => next(true));
      return;
    }
    next(true);
  },

  mounted() {
    this.getSettings();
  },
});
</script>
