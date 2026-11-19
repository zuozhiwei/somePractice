<template>
  <div class="account-page">
    <div class="account-sheet">
      <aside class="account-art" aria-label="技能树学习系统">
        <div class="account-brand"><TrailMark /><span>Skill Tree</span></div>
        <div class="atlas-window" aria-hidden="true"></div>
        <div class="account-name">技能树学习系统</div>
      </aside>
      <section class="account-form" aria-labelledby="account-title">
        <header>
          <h1 id="account-title">{{ title }}</h1>
          <p v-if="description">{{ description }}</p>
        </header>
        <slot />
      </section>
    </div>
  </div>
</template>

<script setup>
import TrailMark from './TrailMark.vue'

defineProps({ title: { type: String, required: true }, description: String })
</script>

<style scoped>
.account-page { min-height: 100vh; min-height: 100dvh; display: grid; place-items: center; padding: clamp(18px, 5vw, 72px); background: var(--color-bg); }
.account-sheet { display: grid; grid-template-columns: minmax(260px, 1fr) minmax(340px, .93fr); width: min(100%, 980px); overflow: hidden; border: 1px solid #cbd4c8; border-radius: 22px; background: var(--color-surface); box-shadow: 0 20px 66px #2443331a; }
.account-art { position: relative; isolation: isolate; display: flex; flex-direction: column; justify-content: space-between; padding: 34px; min-height: 530px; overflow: hidden; color: #fff9e7; background: #183d31; }
.account-art::after { content: ''; position: absolute; z-index: -1; inset: 0; background: linear-gradient(180deg, #183d31 0%, #183d31d6 10%, transparent 33%, transparent 68%, #183d31 99%); }
.account-brand { display: flex; align-items: center; gap: 10px; font: 600 28px/1 Georgia, serif; }
.atlas-window { position: absolute; z-index: -2; inset: 0; background: url('../world/assets/adventure-island.webp') 46% 52% / auto 111% no-repeat; filter: saturate(.74); }
.account-name { align-self: flex-start; border-left: 3px solid var(--color-accent); padding-left: 13px; font-size: 17px; font-weight: 600; letter-spacing: .08em; }
.account-form { align-self: center; width: 100%; padding: clamp(28px, 4.5vw, 62px); }
.account-form header { margin-bottom: 26px; }
.account-form h1 { color: var(--color-text); font-size: clamp(26px, 2.7vw, 35px); line-height: 1.25; letter-spacing: -.04em; }
.account-form header p { font-size: 14px; line-height: 1.55; color: var(--color-text-secondary); margin-top: 7px; }
.account-form :deep(.el-form-item__label) { color: #3f5547; font-weight: 600; font-size: 13px; line-height: 1.4; margin-bottom: 7px; }
.account-form :deep(.el-form-item) { margin-bottom: 21px; }
.account-form :deep(.el-input__wrapper), .account-form :deep(.el-select__wrapper) { min-height: 44px; background: white; }
.account-form :deep(.account-submit) { width: 100%; min-height: 44px; font-size: 15px; font-weight: 650; }
.account-form :deep(.account-links) { display: flex; align-items: center; justify-content: space-between; gap: 7px 14px; flex-wrap: wrap; margin-top: 18px; font-size: 13px; color: var(--color-text-secondary); }
.account-form :deep(.account-links a), .account-form :deep(.account-links button) { display: inline-flex; align-items: center; min-height: 36px; background: none; border: 0; color: var(--color-primary); font-weight: 600; text-decoration: underline; text-underline-offset: 3px; }
.account-form :deep(.invite-banner) { padding: 10px 14px; margin-bottom: 18px; font-size: 13px; line-height: 1.6; background: #fbefd5; color: #775018; border-left: 3px solid #bb832d; }
@media (max-width: 700px) {
  .account-page { padding: 16px; }
  .account-sheet { grid-template-columns: 1fr; max-width: 460px; border-radius: 15px; }
  .account-art { padding: 16px 22px; min-height: auto; }
  .atlas-window { background: none; }
  .account-art::after { display: none; }
  .account-brand { font-size: 23px; gap: 8px; }
  .account-brand .trail-mark { width: 31px; height: 31px; }
  .account-name { position: absolute; right: 22px; align-self: flex-end; top: 25px; font-size: 11px; line-height: 1; border: 0; color: #d6e2d4; }
  .account-form { padding: 27px 22px; }
}
</style>
