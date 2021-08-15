<template>
  <section class="settings">
    <b-loading :is-full-page="true" v-if="loading.settings || isLoading" active />
    <header class="columns">
      <div class="column is-half">
        <h1 class="title is-4">{{ $t('settings.title') }}
          <span class="has-text-grey-light">({{ serverConfig.version }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button :disabled="!hasFormChanged"
          type="is-primary" icon-left="content-save-outline"
          @click="onSubmit" class="isSaveEnabled" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </div>
    </header>
    <hr />

    <section class="wrap-small">
      <form @submit.prevent="onSubmit">
        <b-tabs type="is-boxed" :animated="false">
          <b-tab-item :label="$t('settings.general.name')" label-position="on-border">
            <div class="items">
              <b-field :label="$t('settings.general.rootURL')" label-position="on-border"
                :message="$t('settings.general.rootURLHelp')">
                <b-input v-model="form['app.root_url']" name="app.root_url"
                    placeholder='https://listmonk.yoursite.com' :maxlength="300" />
              </b-field>

              <b-field :label="$t('settings.general.logoURL')" label-position="on-border"
                :message="$t('settings.general.logoURLHelp')">
                <b-input v-model="form['app.logo_url']" name="app.logo_url"
                    placeholder='https://listmonk.yoursite.com/logo.png' :maxlength="300" />
              </b-field>

              <b-field :label="$t('settings.general.faviconURL')" label-position="on-border"
                :message="$t('settings.general.faviconURLHelp')">
                <b-input v-model="form['app.favicon_url']" name="app.favicon_url"
                    placeholder='https://listmonk.yoursite.com/favicon.png' :maxlength="300" />
              </b-field>

              <hr />
              <b-field :label="$t('settings.general.fromEmail')" label-position="on-border"
                :message="$t('settings.general.fromEmailHelp')">
                <b-input v-model="form['app.from_email']" name="app.from_email"
                    placeholder='Listmonk <noreply@listmonk.yoursite.com>'
                    pattern="(.+?)\s<(.+?)@(.+?)>" :maxlength="300" />
              </b-field>

              <b-field :label="$t('settings.general.adminNotifEmails')" label-position="on-border"
                :message="$t('settings.general.adminNotifEmailsHelp')">
                <b-taginput v-model="form['app.notify_emails']" name="app.notify_emails"
                  :before-adding="(v) => v.match(/(.+?)@(.+?)/)"
                  placeholder='you@yoursite.com' />
              </b-field>

              <b-field :label="$t('settings.general.enablePublicSubPage')"
                :message="$t('settings.general.enablePublicSubPageHelp')">
                <b-switch v-model="form['app.enable_public_subscription_page']"
                    name="app.enable_public_subscription_page" />
              </b-field>

              <b-field :label="$t('settings.general.checkUpdates')"
                :message="$t('settings.general.checkUpdatesHelp')">
                <b-switch v-model="form['app.check_updates']"
                    name="app.check_updates" />
              </b-field>

              <hr />
              <b-field :label="$t('settings.general.language')" label-position="on-border">
                <b-select v-model="form['app.lang']" name="app.lang">
                    <option v-for="l in serverConfig.langs" :key="l.code" :value="l.code">
                      {{ l.name }}
                    </option>
                </b-select>
              </b-field>
            </div>
          </b-tab-item><!-- general -->

          <b-tab-item :label="$t('settings.performance.name')">
            <div class="items">
              <b-field :label="$t('settings.performance.concurrency')" label-position="on-border"
                :message="$t('settings.performance.concurrencyHelp')">
                <b-numberinput v-model="form['app.concurrency']"
                    name="app.concurrency" type="is-light"
                    placeholder="5" min="1" max="10000" />
              </b-field>

              <b-field :label="$t('settings.performance.messageRate')" label-position="on-border"
                :message="$t('settings.performance.messageRateHelp')">
                <b-numberinput v-model="form['app.message_rate']"
                    name="app.message_rate" type="is-light"
                    placeholder="5" min="1" max="100000" />
              </b-field>

              <b-field :label="$t('settings.performance.batchSize')" label-position="on-border"
                :message="$t('settings.performance.batchSizeHelp')">
                <b-numberinput v-model="form['app.batch_size']"
                    name="app.batch_size" type="is-light"
                    placeholder="1000" min="1" max="100000" />
              </b-field>

              <b-field :label="$t('settings.performance.maxErrThreshold')"
                label-position="on-border"
                :message="$t('settings.performance.maxErrThresholdHelp')">
                <b-numberinput v-model="form['app.max_send_errors']"
                    name="app.max_send_errors" type="is-light"
                    placeholder="1999" min="0" max="100000" />
              </b-field>

              <div>
                <div class="columns">
                  <div class="column is-6">
                    <b-field :label="$t('settings.performance.slidingWindow')"
                      :message="$t('settings.performance.slidingWindowHelp')">
                      <b-switch v-model="form['app.message_sliding_window']"
                          name="app.message_sliding_window" />
                    </b-field>
                  </div>

                  <div class="column is-3"
                    :class="{'disabled': !form['app.message_sliding_window']}">
                    <b-field :label="$t('settings.performance.slidingWindowRate')"
                      label-position="on-border"
                      :message="$t('settings.performance.slidingWindowRateHelp')">

                      <b-numberinput v-model="form['app.message_sliding_window_rate']"
                        name="sliding_window_rate" type="is-light"
                        controls-position="compact"
                        :disabled="!form['app.message_sliding_window']"
                        placeholder="25" min="1" max="10000000" />
                    </b-field>
                  </div>

                  <div class="column is-3"
                    :class="{'disabled': !form['app.message_sliding_window']}">
                    <b-field :label="$t('settings.performance.slidingWindowDuration')"
                      label-position="on-border"
                      :message="$t('settings.performance.slidingWindowDurationHelp')">

                      <b-input v-model="form['app.message_sliding_window_duration']"
                        name="sliding_window_duration"
                        :disabled="!form['app.message_sliding_window']"
                        placeholder="1h" :pattern="regDuration" :maxlength="10" />
                    </b-field>
                  </div>
                </div>
              </div><!-- sliding window -->
            </div>
          </b-tab-item><!-- performance -->

          <b-tab-item :label="$t('settings.privacy.name')">
            <div class="items">
              <b-field :label="$t('settings.privacy.individualSubTracking')"
                :message="$t('settings.privacy.individualSubTrackingHelp')">
                <b-switch v-model="form['privacy.individual_tracking']"
                    name="privacy.individual_tracking" />
              </b-field>

              <b-field :label="$t('settings.privacy.listUnsubHeader')"
                :message="$t('settings.privacy.listUnsubHeaderHelp')">
                <b-switch v-model="form['privacy.unsubscribe_header']"
                    name="privacy.unsubscribe_header" />
              </b-field>

              <b-field :label="$t('settings.privacy.allowBlocklist')"
                :message="$t('settings.privacy.allowBlocklistHelp')">
                <b-switch v-model="form['privacy.allow_blocklist']"
                    name="privacy.allow_blocklist" />
              </b-field>

              <b-field :label="$t('settings.privacy.allowExport')"
                :message="$t('settings.privacy.allowExportHelp')">
                <b-switch v-model="form['privacy.allow_export']"
                    name="privacy.allow_export" />
              </b-field>

              <b-field :label="$t('settings.privacy.allowWipe')"
                :message="$t('settings.privacy.allowWipeHelp')">
                <b-switch v-model="form['privacy.allow_wipe']"
                    name="privacy.allow_wipe" />
              </b-field>
            </div>
          </b-tab-item><!-- privacy -->

          <b-tab-item :label="$t('settings.media.title')">
            <div class="items">
              <b-field :label="$t('settings.media.provider')" label-position="on-border">
                <b-select v-model="form['upload.provider']" name="upload.provider">
                  <option value="filesystem">filesystem</option>
                  <option value="s3">s3</option>
                </b-select>
              </b-field>

              <div class="block" v-if="form['upload.provider'] === 'filesystem'">
                <b-field :label="$t('settings.media.upload.path')" label-position="on-border"
                  :message="$t('settings.media.upload.pathHelp')">
                  <b-input v-model="form['upload.filesystem.upload_path']"
                      name="app.upload_path" placeholder='/home/listmonk/uploads'
                      :maxlength="200" />
                </b-field>

                <b-field :label="$t('settings.media.upload.uri')" label-position="on-border"
                  :message="$t('settings.media.upload.uriHelp')">
                  <b-input v-model="form['upload.filesystem.upload_uri']"
                      name="app.upload_uri" placeholder='/uploads' :maxlength="200" />
                </b-field>
              </div><!-- filesystem -->

              <div class="block" v-if="form['upload.provider'] === 's3'">
                <div class="columns">
                  <div class="column is-3">
                    <b-field :label="$t('settings.media.s3.region')"
                      label-position="on-border" expanded>
                      <b-input v-model="form['upload.s3.aws_default_region']" @input="onS3URLChange"
                          name="upload.s3.aws_default_region"
                          :maxlength="200" placeholder="ap-south-1" />
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field grouped>
                      <b-field :label="$t('settings.media.s3.key')"
                        label-position="on-border" expanded>
                        <b-input v-model="form['upload.s3.aws_access_key_id']"
                            name="upload.s3.aws_access_key_id" :maxlength="200" />
                      </b-field>
                      <b-field :label="$t('settings.media.s3.secret')"
                        label-position="on-border" expanded
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
                    <b-field :label="$t('settings.media.s3.bucketType')" label-position="on-border">
                      <b-select v-model="form['upload.s3.bucket_type']"
                        name="upload.s3.bucket_type" expanded>
                        <option value="private">
                          {{ $t('settings.media.s3.bucketTypePrivate') }}
                        </option>
                        <option value="public">
                          {{ $t('settings.media.s3.bucketTypePublic') }}
                        </option>
                      </b-select>
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field grouped>
                      <b-field :label="$t('settings.media.s3.bucket')"
                        label-position="on-border" expanded>
                        <b-input v-model="form['upload.s3.bucket']" @input="onS3URLChange"
                            name="upload.s3.bucket" :maxlength="200" placeholder="" />
                      </b-field>
                      <b-field :label="$t('settings.media.s3.bucketPath')"
                        label-position="on-border"
                        :message="$t('settings.media.s3.bucketPathHelp')" expanded>
                        <b-input v-model="form['upload.s3.bucket_path']"
                            name="upload.s3.bucket_path" :maxlength="200" placeholder="/" />
                      </b-field>
                    </b-field>
                  </div>
                </div>
                <div class="columns">
                  <div class="column is-3">
                    <b-field :label="$t('settings.media.s3.uploadExpiry')"
                      label-position="on-border"
                      :message="$t('settings.media.s3.uploadExpiryHelp')" expanded>
                      <b-input v-model="form['upload.s3.expiry']"
                        name="upload.s3.expiry"
                        placeholder="14d" :pattern="regDuration" :maxlength="10" />
                    </b-field>
                  </div>
                  <div class="column">
                    <b-field :label="$t('settings.media.s3.url')"
                      label-position="on-border"
                      :message="$t('settings.media.s3.urlHelp')" expanded>
                      <b-input v-model="form['upload.s3.url']"
                        name="upload.s3.url"
                        :disabled="!form['upload.s3.bucket']"
                        placeholder="https://s3.region.amazonaws.com" :maxlength="200" />
                    </b-field>
                  </div>
                </div>
              </div><!-- s3 -->
            </div>
          </b-tab-item><!-- media -->

          <b-tab-item :label="$t('settings.smtp.name')">
            <div class="items mail-servers">
              <div class="block box" v-for="(item, n) in form.smtp" :key="n">
                <div class="columns">
                  <div class="column is-2">
                    <b-field :label="$t('globals.buttons.enabled')">
                      <b-switch v-model="item.enabled" name="enabled"
                          :native-value="true" data-cy="btn-enable-smtp" />
                    </b-field>
                    <b-field v-if="form.smtp.length > 1">
                      <a @click.prevent="$utils.confirm(null, () => removeSMTP(n))"
                        href="#" class="is-size-7" data-cy="btn-delete-smtp">
                        <b-icon icon="trash-can-outline" size="is-small" />
                        {{ $t('globals.buttons.delete') }}
                      </a>
                    </b-field>
                  </div><!-- first column -->

                  <div class="column" :class="{'disabled': !item.enabled}">
                    <div class="columns">
                      <div class="column is-8">
                        <b-field :label="$t('settings.mailserver.host')" label-position="on-border"
                          :message="$t('settings.mailserver.hostHelp')">
                          <b-input v-model="item.host" name="host"
                            placeholder='smtp.yourmailserver.net' :maxlength="200" />
                        </b-field>
                      </div>
                      <div class="column">
                        <b-field :label="$t('settings.mailserver.port')" label-position="on-border"
                          :message="$t('settings.mailserver.portHelp')">
                          <b-numberinput v-model="item.port" name="port" type="is-light"
                              controls-position="compact"
                              placeholder="25" min="1" max="65535" />
                        </b-field>
                      </div>
                    </div><!-- host -->

                    <div class="columns">
                      <div class="column is-2">
                        <b-field :label="$t('settings.mailserver.authProtocol')"
                          label-position="on-border">
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
                          <b-field :label="$t('settings.mailserver.username')"
                            label-position="on-border" expanded>
                            <b-input v-model="item.username"
                              :disabled="item.auth_protocol === 'none'"
                              name="username" placeholder="mysmtp" :maxlength="200" />
                          </b-field>
                          <b-field :label="$t('settings.mailserver.password')"
                            label-position="on-border" expanded
                            :message="$t('settings.mailserver.passwordHelp')">
                            <b-input v-model="item.password"
                              :disabled="item.auth_protocol === 'none'"
                              name="password" type="password"
                              :placeholder="$t('settings.mailserver.passwordHelp')"
                              :maxlength="200" />
                          </b-field>
                        </b-field>
                      </div>
                    </div><!-- auth -->
                    <hr />

                    <div class="columns">
                      <div class="column is-6">
                        <b-field :label="$t('settings.smtp.heloHost')" label-position="on-border"
                          :message="$t('settings.smtp.heloHostHelp')">
                          <b-input v-model="item.hello_hostname"
                            name="hello_hostname" placeholder="" :maxlength="200" />
                        </b-field>
                      </div>
                      <div class="column">
                        <b-field grouped>
                          <b-field :label="$t('settings.mailserver.tls')" expanded
                            :message="$t('settings.mailserver.tlsHelp')">
                            <b-switch v-model="item.tls_enabled" name="item.tls_enabled" />
                          </b-field>
                          <b-field :label="$t('settings.mailserver.skipTLS')" expanded
                            :message="$t('settings.mailserver.skipTLSHelp')">
                            <b-switch v-model="item.tls_skip_verify"
                              :disabled="!item.tls_enabled" name="item.tls_skip_verify" />
                          </b-field>
                        </b-field>
                      </div>
                    </div><!-- TLS -->
                    <hr />

                    <div class="columns">
                      <div class="column is-3">
                        <b-field :label="$t('settings.mailserver.maxConns')"
                          label-position="on-border"
                          :message="$t('settings.mailserver.maxConnsHelp')">
                          <b-numberinput v-model="item.max_conns" name="max_conns" type="is-light"
                              controls-position="compact"
                              placeholder="25" min="1" max="65535" />
                        </b-field>
                      </div>
                      <div class="column is-3">
                        <b-field :label="$t('settings.smtp.retries')" label-position="on-border"
                          :message="$t('settings.smtp.retriesHelp')">
                          <b-numberinput v-model="item.max_msg_retries" name="max_msg_retries"
                              type="is-light"
                              controls-position="compact"
                              placeholder="2" min="1" max="1000" />
                        </b-field>
                      </div>
                      <div class="column is-3">
                        <b-field :label="$t('settings.mailserver.idleTimeout')"
                          label-position="on-border"
                          :message="$t('settings.mailserver.idleTimeoutHelp')">
                          <b-input v-model="item.idle_timeout" name="idle_timeout"
                            placeholder="15s" :pattern="regDuration" :maxlength="10" />
                        </b-field>
                      </div>
                      <div class="column is-3">
                        <b-field :label="$t('settings.mailserver.waitTimeout')"
                          label-position="on-border"
                          :message="$t('settings.mailserver.waitTimeoutHelp')">
                          <b-input v-model="item.wait_timeout" name="wait_timeout"
                            placeholder="5s" :pattern="regDuration" :maxlength="10" />
                        </b-field>
                      </div>
                    </div>
                    <hr />

                    <div>
                      <p v-if="item.email_headers.length === 0 && !item.showHeaders">
                        <a href="#" class="is-size-7" @click.prevent="() => showSMTPHeaders(n)">
                          <b-icon icon="plus" />{{ $t('settings.smtp.setCustomHeaders') }}</a>
                      </p>
                      <b-field v-if="item.email_headers.length > 0 || item.showHeaders"
                        :label="$t('')" label-position="on-border"
                        :message="$t('settings.smtp.customHeadersHelp')">
                        <b-input v-model="item.strEmailHeaders" name="email_headers" type="textarea"
                          placeholder='[{"X-Custom": "value"}, {"X-Custom2": "value"}]' />
                      </b-field>
                    </div>
                  </div>
                </div><!-- second container column -->
              </div><!-- block -->
            </div><!-- mail-servers -->

            <b-button @click="addSMTP" icon-left="plus" type="is-primary">
              {{ $t('globals.buttons.addNew') }}
            </b-button>
          </b-tab-item><!-- mail servers -->

          <b-tab-item :label="$t('settings.bounces.name')">
            <div class="columns mb-6">
              <div class="column">
                <b-field :label="$t('settings.bounces.enable')">
                  <b-switch v-model="form['bounce.enabled']" name="bounce.enabled" />
                </b-field>
              </div>
              <div class="column" :class="{'disabled': !form['bounce.enabled']}">
                <b-field :label="$t('settings.bounces.count')" label-position="on-border"
                  :message="$t('settings.bounces.countHelp')">
                  <b-numberinput v-model="form['bounce.count']"
                    name="bounce.count" type="is-light"
                    controls-position="compact" placeholder="3" min="1" max="1000" />
                </b-field>
              </div>
              <div class="column" :class="{'disabled': !form['bounce.enabled']}">
                <b-field :label="$t('settings.bounces.action')" label-position="on-border">
                  <b-select name="bounce.action" v-model="form['bounce.action']">
                    <option value="blocklist">{{ $t('settings.bounces.blocklist') }}</option>
                    <option value="delete">{{ $t('settings.bounces.delete') }}</option>
                  </b-select>
                </b-field>
              </div>
            </div><!-- columns -->

            <div class="mb-6">
              <b-field :label="$t('settings.bounces.enableWebhooks')">
                <b-switch v-model="form['bounce.webhooks_enabled']"
                  :disabled="!form['bounce.enabled']"
                  name="webhooks_enabled" :native-value="true"
                  data-cy="btn-enable-bounce-webhook" />
                <p class="has-text-grey">
                  <a href="" target="_blank">{{ $t('globals.buttons.learnMore') }} &rarr;</a>
                </p>
              </b-field>
              <div class="box" v-if="form['bounce.webhooks_enabled']">
                  <div class="columns">
                    <div class="column">
                      <b-field :label="$t('settings.bounces.enableSES')">
                        <b-switch v-model="form['bounce.ses_enabled']"
                          name="ses_enabled" :native-value="true" data-cy="btn-enable-bounce-ses" />
                      </b-field>
                    </div>
                  </div>
                  <div class="columns">
                    <div class="column is-3">
                      <b-field :label="$t('settings.bounces.enableSendgrid')">
                        <b-switch v-model="form['bounce.sendgrid_enabled']"
                          name="sendgrid_enabled" :native-value="true"
                          data-cy="btn-enable-bounce-sendgrid" />
                      </b-field>
                    </div>
                    <div class="column">
                      <b-field :label="$t('settings.bounces.sendgridKey')"
                        :message="$t('globals.messages.passwordChange')">
                        <b-input v-model="form['bounce.sendgrid_key']" type="password"
                          :disabled="!form['bounce.sendgrid_enabled']"
                          name="sendgrid_enabled" :native-value="true"
                          data-cy="btn-enable-bounce-sendgrid" />
                      </b-field>
                    </div>
                  </div>
              </div>
            </div>

            <!-- bounce mailbox -->
            <b-field :label="$t('settings.bounces.enableMailbox')">
              <b-switch v-if="form['bounce.mailboxes']"
                v-model="form['bounce.mailboxes'][0].enabled"
                :disabled="!form['bounce.enabled']"
                name="enabled" :native-value="true" data-cy="btn-enable-bounce-mailbox" />
            </b-field>

            <template v-if="form['bounce.enabled'] && form['bounce.mailboxes'][0].enabled">
              <div class="block box" v-for="(item, n) in form['bounce.mailboxes']" :key="n">
                <div class="columns">
                  <div class="column" :class="{'disabled': !item.enabled}">
                    <div class="columns">
                      <div class="column is-3">
                        <b-field :label="$t('settings.bounces.type')" label-position="on-border">
                          <b-select v-model="item.type" name="type">
                              <option value="pop">POP</option>
                          </b-select>
                        </b-field>
                      </div>
                      <div class="column is-6">
                        <b-field :label="$t('settings.mailserver.host')" label-position="on-border"
                          :message="$t('settings.mailserver.hostHelp')">
                          <b-input v-model="item.host" name="host"
                            placeholder='bounce.yourmailserver.net' :maxlength="200" />
                        </b-field>
                      </div>
                      <div class="column is-3">
                        <b-field :label="$t('settings.mailserver.port')" label-position="on-border"
                          :message="$t('settings.mailserver.portHelp')">
                          <b-numberinput v-model="item.port" name="port" type="is-light"
                              controls-position="compact"
                              placeholder="25" min="1" max="65535" />
                        </b-field>
                      </div>
                    </div><!-- host -->

                    <div class="columns">
                      <div class="column is-3">
                        <b-field :label="$t('settings.mailserver.authProtocol')"
                          label-position="on-border">
                          <b-select v-model="item.auth_protocol" name="auth_protocol">
                            <option value="none">none</option>
                            <option v-if="item.type === 'pop'" value="userpass">userpass</option>
                            <template v-else>
                              <option value="cram">cram</option>
                              <option value="plain">plain</option>
                              <option value="login">login</option>
                            </template>
                          </b-select>
                        </b-field>
                      </div>
                      <div class="column">
                        <b-field grouped>
                          <b-field :label="$t('settings.mailserver.username')"
                            label-position="on-border" expanded>
                            <b-input v-model="item.username"
                              :disabled="item.auth_protocol === 'none'"
                              name="username" placeholder="mysmtp" :maxlength="200" />
                          </b-field>
                          <b-field :label="$t('settings.mailserver.password')"
                            label-position="on-border" expanded
                            :message="$t('settings.mailserver.passwordHelp')">
                            <b-input v-model="item.password"
                              :disabled="item.auth_protocol === 'none'"
                              name="password" type="password"
                              :placeholder="$t('settings.mailserver.passwordHelp')"
                              :maxlength="200" />
                          </b-field>
                        </b-field>
                      </div>
                    </div><!-- auth -->

                    <div class="columns">
                      <div class="column is-6">
                        <b-field grouped>
                          <b-field :label="$t('settings.mailserver.tls')" expanded
                            :message="$t('settings.mailserver.tlsHelp')">
                            <b-switch v-model="item.tls_enabled" name="item.tls_enabled" />
                          </b-field>
                          <b-field :label="$t('settings.mailserver.skipTLS')" expanded
                            :message="$t('settings.mailserver.skipTLSHelp')">
                            <b-switch v-model="item.tls_skip_verify"
                              :disabled="!item.tls_enabled" name="item.tls_skip_verify" />
                          </b-field>
                        </b-field>
                      </div>
                      <div class="column"></div>
                      <div class="column is-4">
                        <b-field :label="$t('settings.bounces.scanInterval')" expanded
                          label-position="on-border"
                          :message="$t('settings.bounces.scanIntervalHelp')">
                          <b-input v-model="item.scan_interval" name="scan_interval"
                            placeholder="15m" :pattern="regDuration" :maxlength="10" />
                        </b-field>
                      </div>
                    </div><!-- TLS -->
                  </div>
                </div><!-- second container column -->
              </div><!-- block -->
            </template>
          </b-tab-item><!-- bounces -->

          <b-tab-item :label="$t('settings.messengers.name')">
            <div class="items messengers">
              <div class="block box" v-for="(item, n) in form.messengers" :key="n">
                <div class="columns">
                  <div class="column is-2">
                    <b-field :label="$t('globals.buttons.enabled')">
                      <b-switch v-model="item.enabled" name="enabled"
                          :native-value="true" />
                    </b-field>
                    <b-field>
                      <a @click.prevent="$utils.confirm(null, () => removeMessenger(n))"
                        href="#" class="is-size-7">
                        <b-icon icon="trash-can-outline" size="is-small" />
                        {{ $t('globals.buttons.delete') }}
                      </a>
                    </b-field>
                  </div><!-- first column -->

                  <div class="column" :class="{'disabled': !item.enabled}">
                    <div class="columns">
                      <div class="column is-4">
                        <b-field :label="$t('globals.fields.name')" label-position="on-border"
                          :message="$t('settings.messengers.nameHelp')">
                          <b-input v-model="item.name" name="name"
                            placeholder='mymessenger' :maxlength="200" />
                        </b-field>
                      </div>
                      <div class="column is-8">
                        <b-field :label="$t('settings.messengers.url')" label-position="on-border"
                          :message="$t('settings.messengers.urlHelp')">
                          <b-input v-model="item.root_url" name="root_url"
                            placeholder='https://postback.messenger.net/path' :maxlength="200" />
                        </b-field>
                      </div>
                    </div><!-- host -->

                    <div class="columns">
                      <div class="column">
                        <b-field grouped>
                          <b-field :label="$t('settings.messengers.username')"
                            label-position="on-border" expanded>
                            <b-input v-model="item.username" name="username" :maxlength="200" />
                          </b-field>
                          <b-field :label="$t('settings.messengers.password')"
                            label-position="on-border" expanded
                            :message="$t('globals.messages.passwordChange')">
                            <b-input v-model="item.password"
                              name="password" type="password"
                              :placeholder="$t('globals.messages.passwordChange')"
                              :maxlength="200" />
                          </b-field>
                        </b-field>
                      </div>
                    </div><!-- auth -->
                    <hr />

                    <div class="columns">
                      <div class="column is-4">
                        <b-field :label="$t('settings.messengers.maxConns')"
                          label-position="on-border"
                          :message="$t('settings.messengers.maxConnsHelp')">
                          <b-numberinput v-model="item.max_conns" name="max_conns" type="is-light"
                              controls-position="compact"
                              placeholder="25" min="1" max="65535" />
                        </b-field>
                      </div>
                      <div class="column is-4">
                        <b-field :label="$t('settings.messengers.retries')"
                          label-position="on-border"
                          :message="$t('settings.messengers.retriesHelp')">
                          <b-numberinput v-model="item.max_msg_retries" name="max_msg_retries"
                              type="is-light"
                              controls-position="compact"
                              placeholder="2" min="1" max="1000" />
                        </b-field>
                      </div>
                      <div class="column is-4">
                        <b-field :label="$t('settings.messengers.timeout')"
                          label-position="on-border"
                          :message="$t('settings.messengers.timeoutHelp')">
                          <b-input v-model="item.timeout" name="timeout"
                            placeholder="5s" :pattern="regDuration" :maxlength="10" />
                        </b-field>
                      </div>
                    </div>
                    <hr />
                  </div>
                </div><!-- second container column -->
              </div><!-- block -->
            </div><!-- mail-servers -->

            <b-button @click="addMessenger" icon-left="plus" type="is-primary">
              {{ $t('globals.buttons.addNew') }}
            </b-button>
          </b-tab-item><!-- messengers -->
        </b-tabs>

      </form>
    </section>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

const dummyPassword = ' '.repeat(8);

export default Vue.extend({
  data() {
    return {
      regDuration: '[0-9]+(ms|s|m|h|d)',
      isLoading: false,

      // formCopy is a stringified copy of the original settings against which
      // form is compared to detect changes.
      formCopy: '',
      form: {},
    };
  },

  methods: {
    addSMTP() {
      this.form.smtp.push({
        enabled: true,
        host: '',
        hello_hostname: '',
        port: 587,
        auth_protocol: 'none',
        username: '',
        password: '',
        email_headers: [],
        max_conns: 10,
        max_msg_retries: 2,
        idle_timeout: '15s',
        wait_timeout: '5s',
        tls_enabled: true,
        tls_skip_verify: false,
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.mail-servers input[name="host"]');
        items[items.length - 1].focus();
      });
    },

    removeSMTP(i) {
      this.form.smtp.splice(i, 1);
    },

    removeBounceBox(i) {
      this.form['bounce.mailboxes'].splice(i, 1);
    },

    showSMTPHeaders(i) {
      const s = this.form.smtp[i];
      s.showHeaders = true;
      this.form.smtp.splice(i, 1, s);
    },

    addMessenger() {
      this.form.messengers.push({
        enabled: true,
        root_url: '',
        name: '',
        username: '',
        password: '',
        max_conns: 25,
        max_msg_retries: 2,
        timeout: '5s',
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.messengers input[name="name"]');
        items[items.length - 1].focus();
      });
    },

    removeMessenger(i) {
      this.form.messengers.splice(i, 1);
    },

    onS3URLChange() {
      // If a custom non-AWS URL has been entered, don't update it automatically.
      if (this.form['upload.s3.url'] !== '' && !this.form['upload.s3.url'].match(/amazonaws\.com/)) {
        return;
      }
      this.form['upload.s3.url'] = `https://s3.${this.form['upload.s3.aws_default_region']}.amazonaws.com`;
    },

    onSubmit() {
      const form = JSON.parse(JSON.stringify(this.form));

      // SMTP boxes.
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

      // Bounces boxes.
      for (let i = 0; i < form['bounce.mailboxes'].length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form['bounce.mailboxes'][i].password === dummyPassword) {
          form['bounce.mailboxes'][i].password = '';
        }
      }

      if (form['upload.s3.aws_secret_access_key'] === dummyPassword) {
        form['upload.s3.aws_secret_access_key'] = '';
      }

      if (form['bounce.sendgrid_key'] === dummyPassword) {
        form['bounce.sendgrid_key'] = '';
      }

      for (let i = 0; i < form.messengers.length; i += 1) {
        // If it's the dummy UI password placeholder, ignore it.
        if (form.messengers[i].password === dummyPassword) {
          form.messengers[i].password = '';
        }
      }

      this.isLoading = true;
      this.$api.updateSettings(form).then((data) => {
        if (data.needsRestart) {
          // There are running campaigns and the app didn't auto restart.
          // The UI will show a warning.
          this.$root.loadConfig();
          this.getSettings();
          this.isLoading = false;
          return;
        }

        this.$utils.toast(this.$t('settings.messengers.messageSaved'));

        // Poll until there's a 200 response, waiting for the app
        // to restart and come back up.
        const pollId = setInterval(() => {
          this.$api.getHealth().then(() => {
            clearInterval(pollId);
            this.$root.loadConfig();
            this.getSettings();
          });
        }, 500);
      }, () => {
        this.isLoading = false;
      });
    },

    getSettings() {
      this.$api.getSettings().then((data) => {
        const d = JSON.parse(JSON.stringify(data));

        // Serialize the `email_headers` array map to display on the form.
        for (let i = 0; i < d.smtp.length; i += 1) {
          d.smtp[i].strEmailHeaders = JSON.stringify(d.smtp[i].email_headers, null, 4);

          // The backend doesn't send passwords, so add a dummy so that
          // the password looks filled on the UI.
          d.smtp[i].password = dummyPassword;
        }

        for (let i = 0; i < d['bounce.mailboxes'].length; i += 1) {
          // The backend doesn't send passwords, so add a dummy so that
          // the password looks filled on the UI.
          d['bounce.mailboxes'][i].password = dummyPassword;
        }

        for (let i = 0; i < d.messengers.length; i += 1) {
          // The backend doesn't send passwords, so add a dummy so that it
          // the password looks filled on the UI.
          d.messengers[i].password = dummyPassword;
        }

        if (d['upload.provider'] === 's3') {
          d['upload.s3.aws_secret_access_key'] = dummyPassword;
        }
        d['bounce.sendgrid_key'] = dummyPassword;

        this.form = d;
        this.formCopy = JSON.stringify(d);

        this.$nextTick(() => {
          this.isLoading = false;
        });
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
      this.$utils.confirm(this.$t('settings.messengers.messageDiscard'), () => next(true));
      return;
    }
    next(true);
  },

  mounted() {
    this.getSettings();
  },
});
</script>
