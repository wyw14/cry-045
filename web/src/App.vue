<script setup lang="ts">
import { ref } from 'vue'
const section = ref('材料库')
const sections = ['材料库','申请向导','审核队列','问题协作','证书中心','项目清单','历史追溯']
</script>
<template>
  <main class="shell">
    <header><h1>工业材料合规中心</h1><span>离线演示 · request_id 可追溯</span></header>
    <nav><button v-for="item in sections" :key="item" :class="{active: section===item}" @click="section=item">{{ item }}</button></nav>
    <section class="card"><h2>{{ section }}</h2><p>当前模块展示材料规格、证书有效期、退回问题和审批时间线。</p><div class="metric"><strong>12</strong><span>待处理选型申请</span><strong>3</strong><span>证书即将到期</span></div></section>
  </main>
</template>
<style scoped>
.shell{font-family:system-ui;color:#203044;max-width:1100px;margin:32px auto;padding:0 20px}header{display:flex;justify-content:space-between;align-items:center}nav{display:flex;gap:8px;margin:24px 0;flex-wrap:wrap}button{border:1px solid #c8d3df;background:white;padding:10px 14px;border-radius:7px;cursor:pointer}.active{background:#1769aa;color:white}.card{border:1px solid #d7e0ea;padding:24px;border-radius:12px}.metric{display:flex;gap:14px;align-items:center}.metric strong{font-size:32px;color:#1769aa}
</style>
