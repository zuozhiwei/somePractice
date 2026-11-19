<template>
  <AccountLayout title="登录" description="登录后，从上次的关卡继续。">
    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="login">
      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" name="username" autocomplete="username" placeholder="请输入用户名" size="large" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input v-model="form.password" name="password" type="password" autocomplete="current-password" placeholder="请输入密码" show-password size="large" />
      </el-form-item>
      <el-button type="primary" native-type="submit" :loading="loading" size="large" class="account-submit">登录</el-button>
    </el-form>
    <div class="account-links">
      <span>还没有账号？ <router-link to="/register">注册</router-link></span>
      <router-link to="/forgot-password">忘记密码？</router-link>
    </div>
  </AccountLayout>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'
import { useAuthStore } from '../store/auth'
import AccountLayout from '../components/AccountLayout.vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref(null)
const loading = ref(false)

const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

async function login() {
  if (loading.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid || loading.value) return
  loading.value = true
  try {
    const res = await authApi.login(form)
    authStore.setAuth(res.data.data.token, res.data.data.user)
    ElMessage.success('登录成功')
    router.push('/learn')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
