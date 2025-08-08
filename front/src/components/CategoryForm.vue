<template>
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="80px"
      size="small"
    >
      <el-form-item label="分类名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入分类名称" maxlength="20" />
      </el-form-item>
  
      <el-form-item label="分类类型" prop="categoryType">
        <el-radio-group v-model="form.categoryType">
          <el-radio label="type">类型</el-radio>
          <el-radio label="tag">标签</el-radio>
          <el-radio label="time">时间</el-radio>
        </el-radio-group>
      </el-form-item>
  
      <el-form-item>
        <el-button type="primary" @click="handleSubmit">提交</el-button>
        <el-button @click="handleCancel" style="margin-left: 8px;">取消</el-button>
      </el-form-item>
    </el-form>
  </template>
  
  <script setup>
  import { reactive, ref } from 'vue'
  
  const emit = defineEmits(['submit', 'cancel'])
  
  const formRef = ref(null)
  const form = reactive({
    name: '',
    categoryType: 'type'
  })
  
  const rules = {
    name: [
      { required: true, message: '请输入分类名称', trigger: 'blur' },
      { min: 2, max: 20, message: '长度在2到20个字符', trigger: 'blur' }
    ],
    categoryType: [{ required: true, message: '请选择分类类型', trigger: 'change' }]
  }
  
  const handleSubmit = () => {
    formRef.value.validate((valid) => {
      if (valid) {
        emit('submit', { ...form })
      }
    })
  }
  
  const handleCancel = () => {
    emit('cancel')
  }
  </script>
  
  <style scoped>
  /* 适当调整表单内边距 */
  .el-form {
    padding: 8px 0;
  }
  </style>
  