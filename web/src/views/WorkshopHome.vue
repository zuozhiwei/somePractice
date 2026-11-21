<template>
  <div class="workshop-home">
    <section class="welcome" aria-labelledby="welcome-title">
      <div class="welcome-copy">
        <span class="welcome-kicker">大学生全栈实战</span>
        <h1 id="welcome-title">做个校园小产品<br />让同学真的用上<span>。</span></h1>
        <p>做一个属于自己学校的匿名树洞。从第一条帖子开始，让同学能发心事、写评论、收到回复，最后把你的站点上线。</p>
        <div class="welcome-actions">
          <RouterLink to="/register" class="workshop-button">开始做项目 <ArrowRight aria-hidden="true" /></RouterLink>
          <RouterLink :to="{ path: '/', hash: '#project-route' }" class="hero-secondary">看看项目路线 <ArrowDown aria-hidden="true" /></RouterLink>
        </div>
      </div>
      <div class="welcome-art">
        <StudioIllustration alt="三位同学在校园自习室一起做项目" sizes="(max-width: 720px) 100vw, (max-width: 1304px) 52vw, 644px" priority="high" />
      </div>
    </section>

    <section id="project-route" aria-labelledby="route-title" class="public-route">
      <div class="route-heading">
        <div>
          <h2 id="route-title">校园树洞 · 项目路线</h2>
          <p>从创建文件夹、打开第一张网页开始，逐步加上投稿、评论和回复。</p>
        </div>
        <RouterLink class="text-link" to="/login">继续我的项目 <ArrowRight aria-hidden="true" /></RouterLink>
      </div>
      <ProjectStages />
    </section>
  </div>
</template>

<script setup>
import { ArrowDown, ArrowRight } from '@element-plus/icons-vue'
import ProjectStages from '../workshop/ProjectStages.vue'
import StudioIllustration from '../workshop/StudioIllustration.vue'
</script>

<style scoped>
.workshop-home { width: min(100% - 64px, 1240px); margin: 28px auto 48px; }
.welcome { display: grid; grid-template-columns: minmax(0, .96fr) minmax(0, 1.04fr); overflow: hidden; background: #e9efdf; border: 1px solid #d9e2ce; border-radius: 23px; }
.welcome-copy { padding: clamp(30px, 4.2vw, 62px) clamp(25px, 4vw, 60px); align-self: center; }
.welcome-kicker { display: inline-flex; align-items: center; gap: 8px; color: #416451; font-size: 13px; font-weight: 700; letter-spacing: .12em; }
.welcome-kicker::before { content: ''; background: #668b59; width: 7px; height: 7px; border-radius: 50%; }
.welcome h1 { font-size: clamp(30px, 4.1vw, 60px); line-height: 1.17; letter-spacing: -.052em; margin-top: 15px; }
.welcome h1 span { color: #50744d; }
.welcome p { margin-top: 19px; max-width: 31em; color: #455b4e; font-size: clamp(14px, 1.5vw, 16px); line-height: 1.85; }
.welcome-actions { display: flex; align-items: center; gap: 13px 19px; flex-wrap: wrap; margin-top: 25px; }
.hero-secondary { display: inline-flex; align-items: center; gap: 7px; min-height: 44px; color: #2a523e; font-weight: 650; font-size: 14px; text-decoration: none; }
.hero-secondary:hover { text-decoration: underline; text-underline-offset: 4px; }
.hero-secondary svg { width: 16px; height: 16px; }
.welcome-art { position: relative; min-height: 335px; background: #e8c681; }
.welcome-art { --studio-position: center 52%; }
.public-route { margin-top: 36px; scroll-margin-top: 18px; }
.route-heading { display: flex; justify-content: space-between; gap: 14px; align-items: end; margin-bottom: 16px; }
.route-heading h2 { color: var(--color-text); font-size: clamp(20px, 2.5vw, 27px); letter-spacing: -.037em; }
.route-heading p { color: var(--color-text-secondary); font-size: 14px; margin-top: 6px; line-height: 1.6; }
@media (max-width: 720px) {
  .workshop-home { width: calc(100% - 32px); margin-top: 16px; }
  .welcome { grid-template-columns: 1fr; border-radius: 16px; }
  .welcome-art { min-height: 0; aspect-ratio: 16/8.5; }
  .welcome-copy { padding: 25px 21px; }
  .welcome h1 { font-size: clamp(28px, 8vw, 47px); }
  .welcome p { margin-top: 13px; }
  .welcome-actions { margin-top: 18px; }
  .public-route { margin-top: 23px; }
  .route-heading { flex-wrap: wrap; }
}
</style>
