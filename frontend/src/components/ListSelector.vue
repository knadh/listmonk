<template>
  <div class="field">
    <b-field :label="label  + (selectedItems ? ` (${selectedItems.length})` : '')">
      <div :class="classes">
        <b-taglist>
          <b-tag v-for="l in selectedItems"
            :key="l.id"
            :class="l.subscriptionStatus"
            :closable="!disabled && l.subscriptionStatus !== 'unsubscribed'"
            :data-id="l.id"
            @close="removeList(l.id)">
            {{ l.name }} <sup>{{ l.subscriptionStatus }}</sup>
          </b-tag>
        </b-taglist>
      </div>
    </b-field>

    <b-field :message="message">
      <b-autocomplete
        :placeholder="placeholder"
        clearable
        dropdown-position="top"
        :disabled="disabled || filteredLists.length === 0"
        :keep-first="true"
        :clear-on-select="true"
        :open-on-focus="true"
        :data="filteredLists"
        @select="selectList"
        field="name">
      </b-autocomplete>
    </b-field>
  </div>
</template>

<script>
import Vue from 'vue';

export default {
  name: 'ListSelector',

  props: {
    label: String,
    placeholder: String,
    message: String,
    required: Boolean,
    disabled: Boolean,
    classes: {
      type: Array,
      default: () => [],
    },
    selected: {
      type: Array,
      default: () => [],
    },
    all: {
      type: Array,
      default: () => [],
    },
  },

  data() {
    return {
      selectedItems: [],
    };
  },

  methods: {
    selectList(l) {
      if (!l) {
        return;
      }
      this.selectedItems.push(l);

      // Propagate the items to the parent's v-model binding.
      Vue.nextTick(() => {
        this.$emit('input', this.selectedItems);
      });
    },

    removeList(id) {
      this.selectedItems = this.selectedItems.filter((l) => l.id !== id);

      // Propagate the items to the parent's v-model binding.
      Vue.nextTick(() => {
        this.$emit('input', this.selectedItems);
      });
    },
  },

  computed: {
    // Returns the list of lists to which the subscriber isn't subscribed.
    filteredLists() {
      // Get a map of IDs of the user subsciptions. eg: {1: true, 2: true};
      const subIDs = this.selectedItems.reduce((obj, item) => ({ ...obj, [item.id]: true }), {});

      // Filter lists from the global lists whose IDs are not in the user's
      // subscribed ist.
      return this.$props.all.filter((l) => !(l.id in subIDs));
    },
  },

  watch: {
    // This is required to update the array of lists to propagate from parent
    // components and "react" on the selector.
    selected() {
      // Deep-copy.
      this.selectedItems = JSON.parse(JSON.stringify(this.selected));
    },
  },
};
</script>
