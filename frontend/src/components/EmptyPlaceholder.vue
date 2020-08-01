<template>
    <section class="section">
        <div class="content has-text-grey has-text-centered">
            <p>
                <b-icon :icon="!icon ? 'plus' : icon" size="is-large" />
            </p>
            <p>{{ !label ? 'Nothing here' : label  }}</p>
        </div>
    </section>
</template>


<script>
export default {
  name: 'EmptyPlaceholder',

  props: {
    icon: String,
    label: String,
  },
};
</script>
