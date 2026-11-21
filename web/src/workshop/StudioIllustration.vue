<template>
  <picture class="studio-picture">
    <source v-if="desktopOnly" media="(max-width: 700px)"
            srcset="data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=" />
    <source type="image/avif" :srcset="smallArt + ' 480w, ' + midArt + ' 704w, ' + largeArt + ' 1152w'" :sizes="sizes" />
    <img :src="fallbackArt" :alt="alt" :fetchpriority="priority" decoding="async" width="960" height="720" />
  </picture>
</template>

<script setup>
import smallArt from './assets/campus-studio-480.avif'
import midArt from './assets/campus-studio-704.avif'
import largeArt from './assets/campus-studio-1152.avif'
import fallbackArt from './assets/campus-studio.webp'

defineProps({
  alt: { type: String, default: '' },
  sizes: { type: String, required: true },
  priority: { type: String, default: 'auto' },
  desktopOnly: Boolean
})
</script>

<style scoped>
.studio-picture, .studio-picture img { position: absolute; inset: 0; width: 100%; height: 100%; }
.studio-picture img { object-fit: cover; object-position: var(--studio-position, center); }
</style>
