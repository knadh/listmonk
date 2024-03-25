<template>
  <div class="items">
    <div class="columns">
      <div class="column">
        <b-field :label="$t('settings.media.provider')" label-position="on-border">
          <b-select v-model="data['upload.provider']" name="upload.provider">
            <option value="filesystem">
              filesystem
            </option>
            <option value="s3">
              s3
            </option>
          </b-select>
        </b-field>
      </div>
      <div class="column is-10">
        <b-field :label="$t('settings.media.upload.extensions')" label-position="on-border" expanded>
          <b-taginput v-model="data['upload.extensions']" name="tags" ellipsis icon="tag-outline"
            placeholder="jpg, png, gif .." />
        </b-field>
      </div>
    </div>
    <hr />

    <div class="block" v-if="data['upload.provider'] === 'filesystem'">
      <b-field :label="$t('settings.media.upload.path')" label-position="on-border"
        :message="$t('settings.media.upload.pathHelp')">
        <b-input v-model="data['upload.filesystem.upload_path']" name="app.upload_path"
          placeholder="/home/listmonk/uploads" :maxlength="200" required />
      </b-field>

      <b-field :label="$t('settings.media.upload.uri')" label-position="on-border"
        :message="$t('settings.media.upload.uriHelp')">
        <b-input v-model="data['upload.filesystem.upload_uri']" name="app.upload_uri" placeholder="/uploads"
          :maxlength="200" required pattern="^\/(.+?)" />
      </b-field>
    </div><!-- filesystem -->

    <div class="block" v-if="data['upload.provider'] === 's3'">
      <div class="columns">
        <div class="column is-3">
          <b-field :label="$t('settings.media.s3.region')" label-position="on-border" expanded>
            <b-input v-model="data['upload.s3.aws_default_region']" @input="onS3URLChange"
              name="upload.s3.aws_default_region" :maxlength="200" placeholder="ap-south-1" />
          </b-field>
        </div>
        <div class="column">
          <b-field grouped>
            <b-field :label="$t('settings.media.s3.key')" label-position="on-border" expanded>
              <b-input v-model="data['upload.s3.aws_access_key_id']" name="upload.s3.aws_access_key_id"
                :maxlength="200" />
            </b-field>
            <b-field :label="$t('settings.media.s3.secret')" label-position="on-border" expanded
              message="Enter a value to change.">
              <b-input v-model="data['upload.s3.aws_secret_access_key']" name="upload.s3.aws_secret_access_key"
                type="password" :maxlength="200" />
            </b-field>
          </b-field>
        </div>
      </div>

      <div class="columns">
        <div class="column is-3">
          <b-field :label="$t('settings.media.s3.bucketType')" label-position="on-border">
            <b-select v-model="data['upload.s3.bucket_type']" name="upload.s3.bucket_type" expanded>
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
            <b-field :label="$t('settings.media.s3.bucket')" label-position="on-border" expanded>
              <b-input v-model="data['upload.s3.bucket']" @input="onS3URLChange" name="upload.s3.bucket" :maxlength="200"
                placeholder="" />
            </b-field>
            <b-field :label="$t('settings.media.s3.bucketPath')" label-position="on-border"
              :message="$t('settings.media.s3.bucketPathHelp')" expanded>
              <b-input v-model="data['upload.s3.bucket_path']" name="upload.s3.bucket_path" :maxlength="200"
                placeholder="/" />
            </b-field>
          </b-field>
        </div>
      </div>

      <div class="columns">
        <div class="column is-3">
          <b-field :label="$t('settings.media.s3.uploadExpiry')" label-position="on-border"
            :message="$t('settings.media.s3.uploadExpiryHelp')" expanded>
            <b-input v-model="data['upload.s3.expiry']" name="upload.s3.expiry" placeholder="14d" :pattern="regDuration"
              :maxlength="10" />
          </b-field>
        </div>
        <div class="column is-9">
          <b-field :label="$t('settings.media.s3.url')" label-position="on-border"
            :message="$t('settings.media.s3.urlHelp')">
            <b-input v-model="data['upload.s3.url']" name="upload.s3.url" :disabled="!data['upload.s3.bucket']" required
              placeholder="https://s3.$region.amazonaws.com" :maxlength="200" expanded type="url" pattern="https?://.*" />
          </b-field>
          <b-field :label="$t('settings.media.s3.publicURL')" label-position="on-border" expanded>
            <b-input v-model="data['upload.s3.public_url']" :message="$t('settings.media.s3.publicURLHelp')"
              name="upload.s3.public_url" :disabled="!data['upload.s3.bucket']" placeholder="https://files.yourdomain.com"
              :maxlength="200" type="url" pattern="https?://.*" />
          </b-field>
        </div>
      </div>
    </div><!-- s3 -->
  </div>
</template>

<script>
import Vue from 'vue';
import { regDuration } from '../../constants';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
      regDuration,
      extensions: [],
    };
  },

  methods: {
    onS3URLChange() {
      // If a custom non-AWS URL has been entered, don't update it automatically.
      if (this.data['upload.s3.url'] !== '' && !this.data['upload.s3.url'].match(/amazonaws\.com/)) {
        return;
      }
      this.data['upload.s3.url'] = `https://s3.${this.data['upload.s3.aws_default_region']}.amazonaws.com`;
    },
  },
});
</script>
