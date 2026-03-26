<template>
  <div class="page-header">
    <div class="header-content">
      <div class="header-info">
        <div class="header-icon" :style="{ background: iconBg }">
          <el-icon :size="iconSize" :color="iconColor">
            <component :is="icon" />
          </el-icon>
        </div>
        <div class="header-text">
          <h1 class="page-title">{{ title }}</h1>
          <p v-if="description" class="page-description">{{ description }}</p>
        </div>
      </div>

      <div v-if="stats && stats.length" class="header-stats">
        <div v-for="item in stats" :key="item.label" class="stat-item">
          <span class="stat-number">{{ item.value }}</span>
          <span class="stat-label">{{ item.label }}</span>
        </div>
      </div>

      <div v-else-if="$slots.actions" class="header-actions">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  icon: {
    type: [Object, Function],
    required: true,
  },
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: '',
  },
  stats: {
    type: Array,
    default: () => [],
  },
  iconBg: {
    type: String,
    default: '#e2e8f0',
  },
  iconColor: {
    type: String,
    default: '#ffffff',
  },
  iconSize: {
    type: Number,
    default: 28,
  },
})
</script>

<style scoped>
.page-header {
  background: #f8fafc;
  padding: 6px 32px;
  border-bottom: 1px solid var(--border-light);
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 2px 0;
}

.page-description {
  font-size: 14px;
  opacity: 0.9;
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
}

.header-stats {
  display: flex;
  gap: 24px;
}

.stat-item {
  text-align: center;
}

.stat-number {
  display: block;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}
</style>
