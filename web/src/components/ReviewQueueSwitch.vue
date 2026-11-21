<template>
  <div class="queue-toolbar">
    <el-radio-group :model-value="modelValue" aria-label="审核状态列表"
                    @update:model-value="emit('update:modelValue', $event)" @change="emit('change')">
      <el-radio-button value="pending">待审核</el-radio-button>
      <el-radio-button value="reviewed">已审核</el-radio-button>
    </el-radio-group>
    <small v-if="!busy && total > 0" class="queue-count">{{ total }} 项提交</small>
  </div>
</template>

<script setup>
defineProps({ modelValue: String, total: Number, busy: Boolean })
const emit = defineEmits(['update:modelValue', 'change'])
</script>

<style scoped>
.queue-toolbar { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px; margin: 0 0 12px; }
.queue-count { color: var(--color-text-secondary); font-size: 13px; }
.queue-toolbar :deep(.el-radio-button__inner) { min-height: 36px; display: grid; place-items: center; }
</style>
