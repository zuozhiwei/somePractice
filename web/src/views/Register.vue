<template>
  <AccountLayout title="创建账号">
    <p v-if="isInvited" class="invite-banner">你正在使用师父的邀请链接。注册后会成为内部成员，并自动加入该师父名下。</p>
    <p v-else-if="legacyInvite" class="invite-banner">此邀请链接已过期，请联系师父获取新链接。</p>
    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="register">
      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" name="username" autocomplete="username" placeholder="用于登录" size="large" />
      </el-form-item>
      <el-form-item label="姓名（选填）" prop="name">
        <el-input v-model="form.name" name="name" autocomplete="name" size="large" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input v-model="form.password" name="password" type="password" autocomplete="new-password" placeholder="至少 6 位" show-password size="large" />
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" name="email" type="email" autocomplete="email" placeholder="用于找回密码" size="large" />
      </el-form-item>
      <el-form-item v-if="!isInvited" label="身份" prop="role">
        <el-select v-model="form.role" size="large" style="width:100%">
          <el-option label="自主学习" value="EXTERNAL" />
          <el-option label="加入教学团队（需邀请码）" value="INTERNAL" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="!isInvited && form.role === 'INTERNAL'" label="邀请码" prop="teamInviteCode">
        <el-input v-model="form.teamInviteCode" placeholder="向管理员获取邀请码" size="large" />
      </el-form-item>
      <el-button type="primary" native-type="submit" :loading="loading" size="large" class="account-submit">注册</el-button>
    </el-form>
    <p class="account-links"><span>已有账号？ <router-link :to="{ path: '/login', query: { next: route.query.next } }">登录</router-link></span></p>
  </AccountLayout>
</template>

<script setup>
import { reactive, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { afterSignIn } from '../router'
import { authApi } from '../api/auth'
import { useAuthStore } from '../store/auth'
import AccountLayout from '../components/AccountLayout.vue'
import { ElMessage } from 'element-plus'
import 'element-plus/es/components/message/style/css'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const inviteToken = ref(route.query.invite ? String(route.query.invite) : null)
const legacyInvite = computed(() => !!route.query.mentor && !inviteToken.value)
// 邀请注册：自动成为内部成员，隐藏身份选择与邀请码，只填基本信息
const isInvited = computed(() => !!inviteToken.value)
const formRef = ref(null)
const loading = ref(false)

const form = reactive({
  username: '', name: '', password: '', email: '', role: isInvited.value ? 'INTERNAL' : 'EXTERNAL', inviteToken: inviteToken.value, teamInviteCode: ''
})
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, min: 6, message: '密码至少6位', trigger: 'blur' }],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }, { type: 'email', message: '请输入有效邮箱', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
  teamInviteCode: [{ required: true, message: '请输入邀请码', trigger: 'blur' }]
}

async function register() {
  if (loading.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid || loading.value) return
  loading.value = true
  try {
    const res = await authApi.register(form)
    authStore.setAuth(res.data.data.token, res.data.data.user)
    ElMessage.success('注册成功')
    router.push(afterSignIn(route.query.next))
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '注册失败')
  } finally {
    loading.value = false
  }
}
</script>
