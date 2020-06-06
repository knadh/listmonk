<template>
  <section class="dashboard content">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-5">{{ dayjs().format("ddd, DD MMM") }}</h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="plus" @click="showNewForm">New</b-button>
      </div>
    </header>

    <div class="columns counts">
      <div class="column is-half">
        <div class="level">
          <div class="level-item has-text-centered">
            <div>
              <p class="title">0</p>
              <p class="heading">Subscribers</p>
            </div>
          </div>
          <div class="level-item has-text-centered">
            <div>
              <p class="title">0</p>
              <p class="heading">Lists</p>
            </div>
          </div>
          <div class="level-item has-text-centered">
            <div>
              <p class="title">0</p>
              <p class="heading">Campaigns</p>
            </div>
          </div>
          <div class="level-item has-text-centered">
            <div>
              <p class="title">0</p>
              <p class="heading">Messages sent</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import dayjs from 'dayjs';

export default Vue.extend({
  name: 'Home',

  computed: {
    dayjs() {
      return dayjs;
    },
  },
});
</script>
