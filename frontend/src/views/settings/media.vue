<template>
  <div class="items">
    <div class="row">
      <div class="col-2">
        <oat-field :label="$t('settings.media.provider')">
          <select aria-label="field" v-model="data['upload.provider']" name="upload.provider">
            <option value="filesystem">
              filesystem
            </option>
            <option value="s3">
              s3
            </option>
          </select>
        </oat-field>
      </div>
      <div class="col-10">
        <oat-field :label="$t('settings.media.upload.extensions')">
          <oat-tag-input v-model="data['upload.extensions']" name="tags"
            placeholder="jpg, png, gif .." />
        </oat-field>
      </div>
    </div>
    <hr />

    <div v-if="data['upload.provider'] === 'filesystem'">
      <oat-field :label="$t('settings.media.upload.path')"
        :message="$t('settings.media.upload.pathHelp')">
        <input aria-label="field" v-model="data['upload.filesystem.upload_path']" name="app.upload_path"
          placeholder="/home/listmonk/uploads" :maxlength="200" required>
      </oat-field>

      <oat-field :label="$t('settings.media.upload.uri')"
        :message="$t('settings.media.upload.uriHelp')">
        <input aria-label="field" v-model="data['upload.filesystem.upload_uri']" name="app.upload_uri" placeholder="/uploads"
          :maxlength="200" required pattern="^\/(.+?)">
      </oat-field>
    </div><!-- filesystem -->

    <div v-if="data['upload.provider'] === 's3'">
      <oat-field :label="$t('settings.media.s3.region')">
        <input aria-label="field" v-model="data['upload.s3.aws_default_region']" @input="onS3URLChange"
          name="upload.s3.aws_default_region" :maxlength="200" placeholder="ap-south-1">
      </oat-field>

      <oat-field :label="$t('settings.media.s3.key')">
        <input aria-label="field" v-model="data['upload.s3.aws_access_key_id']" name="upload.s3.aws_access_key_id" :maxlength="200">
      </oat-field>

      <oat-field :label="$t('settings.media.s3.secret')"
        message="Enter a value to change.">
        <input aria-label="field" v-model="data['upload.s3.aws_secret_access_key']" name="upload.s3.aws_secret_access_key"
          type="password" :maxlength="200">
      </oat-field>

      <oat-field :label="$t('settings.media.s3.bucketType')">
        <select aria-label="field" v-model="data['upload.s3.bucket_type']" name="upload.s3.bucket_type">
          <option value="private">
            {{ $t('settings.media.s3.bucketTypePrivate') }}
          </option>
          <option value="public">
            {{ $t('settings.media.s3.bucketTypePublic') }}
          </option>
        </select>
      </oat-field>

      <oat-field :label="$t('settings.media.s3.bucket')">
        <input aria-label="field" v-model="data['upload.s3.bucket']" @input="onS3URLChange" name="upload.s3.bucket" :maxlength="200"
          placeholder="">
      </oat-field>

      <oat-field :label="$t('settings.media.s3.bucketPath')"
        :message="$t('settings.media.s3.bucketPathHelp')">
        <input aria-label="field" v-model="data['upload.s3.bucket_path']" name="upload.s3.bucket_path" :maxlength="200"
          placeholder="/">
      </oat-field>

      <oat-field :label="$t('settings.media.s3.uploadExpiry')"
        :message="$t('settings.media.s3.uploadExpiryHelp')">
        <input aria-label="field" v-model="data['upload.s3.expiry']" name="upload.s3.expiry" placeholder="14d" :pattern="regDuration"
          :maxlength="10">
      </oat-field>

      <oat-field :label="$t('settings.media.s3.url')"
        :message="$t('settings.media.s3.urlHelp')">
        <input aria-label="field" v-model="data['upload.s3.url']" name="upload.s3.url" required
          placeholder="https://s3.$region.amazonaws.com" :maxlength="200" type="url" pattern="https?://.*">
      </oat-field>

      <oat-field :label="$t('settings.media.s3.publicURL')"
        :message="$t('settings.media.s3.publicURLHelp')">
        <input aria-label="field" v-model="data['upload.s3.public_url']" name="upload.s3.public_url"
          placeholder="https://files.yourdomain.com" :maxlength="200" type="string" pattern="(https?://.*|/.+)">
      </oat-field>
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
