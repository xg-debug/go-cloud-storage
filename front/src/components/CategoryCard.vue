<template>
    <div class="category-card" @click="$emit('click')">
        <div class="card-icon">
            <el-icon :size="36">
                <component :is="icon" />
            </el-icon>
        </div>
        <div class="card-info">
            <div class="card-name">{{ name }}</div>
            <div class="card-count">{{ count }}个文件</div>
        </div>
    </div>
</template>
  
<script setup>
defineProps({
    icon: Object,
    name: String,
    count: Number
})
</script>
  
<style scoped>
.category-card {
    padding: 16px;
    margin-bottom: 20px;
    border-radius: 8px;
    background: var(--el-bg-color);
    cursor: pointer;
    transition: all 0.3s;
    border: 1px solid var(--el-border-color);

    &:hover {
        border-color: var(--el-color-primary);
        box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
    }
}

.card-icon {
    text-align: center;
    margin-bottom: 12px;
    color: var(--el-color-primary);
}

.card-info {
    text-align: center;

    .card-name {
        font-weight: 500;
        margin-bottom: 4px;
    }

    .card-count {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }
}
</style>