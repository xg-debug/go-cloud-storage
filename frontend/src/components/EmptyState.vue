<template>
  <div class="cb-empty-state">
    <div class="empty-icon">
      <el-icon :size="36"><component :is="icon" /></el-icon>
    </div>
    <h3>{{ title }}</h3>
    <p>{{ description }}</p>
  </div>
</template>

<script setup>
defineProps({
  icon: { type: [Object, String], required: true },
  title: { type: String, required: true },
  description: { type: String, default: '' }
})
</script>
