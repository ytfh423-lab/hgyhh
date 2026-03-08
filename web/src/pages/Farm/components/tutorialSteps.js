/**
 * 教程步骤配置 v2 — 交互式教学
 *
 * actionType:
 *   'info'         纯说明步骤，手动下一步
 *   'navigate'     导航到指定页面（点击导航项或自动跳转）
 *   'wait-action'  等待真实业务操作成功（监听 tutorialEvents，仅 success:true 推进）
 *
 * 新增字段:
 *   autoScroll     是否自动滚动目标到可视区域 (默认 true)
 *   waitTarget     是否等待 targetSelector 对应 DOM 出现后再激活步骤 (默认 true)
 *   retryable      失败时允许重试 (默认 true)
 */

// ══════════════════════════════════════════════
//  基础教程 (farm_basic) — 首次进入，不可跳过
//  纯引导浏览：介绍各功能区域，不要求真实操作
// ══════════════════════════════════════════════
const farmBasicSteps = [
  // ── 1. 欢迎 ──
  {
    id: 'fb-1', page: 'overview',
    title: '欢迎来到农场！',
    content: '这是你的专属农场 🌾 接下来带你快速了解农场的核心功能，看完就能上手！',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },

  // ── 2. 资源栏介绍 ──
  {
    id: 'fb-2', page: 'overview',
    title: '💰 资源栏',
    content: '上方显示你的金币余额、等级、地块数量和天气。所有操作都会影响这些数值。',
    targetSelector: '.farm-statusbar', placement: 'bottom',
    actionType: 'info',
  },

  // ── 3. 地块介绍 ──
  {
    id: 'fb-3', page: 'overview',
    title: '🌱 你的农田',
    content: '这些是你的地块。空地可以种植作物，种下后会显示生长进度。',
    targetSelector: '.farm-plot-grid', placement: 'top',
    actionType: 'info',
  },

  // ── 4. 快捷操作栏 ──
  {
    id: 'fb-4', page: 'overview',
    title: '⚡ 快捷操作',
    content: '这里是快捷操作栏——浇水、施肥、收获等常用功能都在这里，一键搞定！',
    targetSelector: '[data-tutorial="quick-actions"]', placement: 'bottom',
    actionType: 'info',
  },

  // ── 5. 导航到种植页 ──
  {
    id: 'fb-5', page: 'overview',
    title: '前往种植页',
    content: '点击左侧「🌱 种植」可以进入种植页面，选择作物和地块。',
    targetSelector: '[data-tutorial="nav-plant"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'plant',
  },

  // ── 6. 种植页介绍 ──
  {
    id: 'fb-6', page: 'plant',
    title: '🌱 种植页面',
    content: '在这里选择作物，然后点击空地即可种植。新手推荐 🥔土豆 或 🌾小麦——价格便宜、见效快！',
    targetSelector: '[data-tutorial="crop-grid"]', placement: 'bottom',
    actionType: 'info',
  },

  // ── 7. 回到总览 ──
  {
    id: 'fb-7', page: 'plant',
    title: '回到总览',
    content: '了解完种植页后，回到总览继续看看其他功能。',
    targetSelector: '[data-tutorial="nav-overview"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'overview',
  },

  // ── 8. 核心流程说明 ──
  {
    id: 'fb-8', page: 'overview',
    title: '� 核心流程',
    content: '农场的核心流程是：种植 → 浇水 → 等待成熟 → 收获 → 出售赚钱。作物成熟后点击「🌾 收获出售」即可卖出！',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },

  // ── 9. 完成 ──
  {
    id: 'fb-9', page: 'overview',
    title: '教程完成！🎉',
    content: '恭喜！你已了解农场的基本操作。现在就去种下你的第一颗作物吧！随着等级提升会解锁更多功能。点击右上角「📖」可随时回顾教程。',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },
];

// ══════════════════════════════════════════════
//  树场教程 (treefarm) — 解锁时触发，可跳过
// ══════════════════════════════════════════════
const treefarmSteps = [
  {
    id: 'tf-1', page: 'overview',
    title: '🌲 树场已解锁！',
    content: '恭喜！树场是长期投资——种下树木，定期收获果实或伐木获取大量资源。',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },
  {
    id: 'tf-2', page: 'overview',
    title: '进入树场',
    content: '点击左侧「🌲 树场」进入看看。',
    targetSelector: '[data-tutorial="nav-treefarm"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'treefarm',
  },
  {
    id: 'tf-3', page: 'treefarm',
    title: '🌲 林地总览',
    content: '每个空树位可种一棵树。树木生长较慢但产出价值高，可以浇水加速。新手推荐普通木材树。',
    targetSelector: '.tree-farm-grid', placement: 'bottom',
    actionType: 'info',
  },
];

// ══════════════════════════════════════════════
//  市场教程 (market) — 解锁时触发，可跳过
// ══════════════════════════════════════════════
const marketSteps = [
  {
    id: 'mk-1', page: 'overview',
    title: '📈 市场已解锁！',
    content: '市场价格实时波动，学会观察趋势可以获得更高收益。',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },
  {
    id: 'mk-2', page: 'overview',
    title: '进入市场',
    content: '点击左侧「📈 市场」查看实时行情。',
    targetSelector: '[data-tutorial="nav-market"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'market',
  },
  {
    id: 'mk-3', page: 'market',
    title: '市场行情',
    content: '绿色箭头表示涨价，红色表示跌价。在高价时出售可以赚更多！',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },
];

// ══════════════════════════════════════════════
//  仓库教程 (warehouse) — 可跳过
// ══════════════════════════════════════════════
const warehouseSteps = [
  {
    id: 'wh-1', page: 'overview',
    title: '📦 仓库系统',
    content: '仓库暂存收获物，等市场价格高时再出售。注意物品有保质期！',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },
];

// ══════════════════════════════════════════════
//  任务教程 (tasks) — 可跳过
// ══════════════════════════════════════════════
const tasksSteps = [
  {
    id: 'tk-1', page: 'overview',
    title: '📝 每日任务',
    content: '每天刷新新任务，完成后点击「领取」获得金币和经验。记得每天登录！',
    targetSelector: null, placement: 'center',
    actionType: 'info',
  },
];

// ══════════════════════════════════════════════
//  汇总导出
// ══════════════════════════════════════════════
export const tutorialFlows = {
  farm_basic: {
    featureKey: 'farm_basic',
    label: '基础教程',
    emoji: '🌾',
    steps: farmBasicSteps,
    forcedOnFirstEntry: true,
  },
  treefarm: {
    featureKey: 'treefarm',
    label: '树场教程',
    emoji: '🌲',
    steps: treefarmSteps,
    unlockLevel: 5,
    skippable: true,
  },
  market: {
    featureKey: 'market',
    label: '市场教程',
    emoji: '📈',
    steps: marketSteps,
    unlockLevel: 2,
    skippable: true,
  },
  warehouse: {
    featureKey: 'warehouse',
    label: '仓库教程',
    emoji: '📦',
    steps: warehouseSteps,
    unlockLevel: 1,
    skippable: true,
  },
  tasks: {
    featureKey: 'tasks',
    label: '任务教程',
    emoji: '📝',
    steps: tasksSteps,
    unlockLevel: 1,
    skippable: true,
  },
};

export function getFlowSteps(flowKey) {
  return tutorialFlows[flowKey]?.steps || [];
}

export function getUnlockableFlows(userLevel) {
  return Object.entries(tutorialFlows)
    .filter(([, flow]) => flow.unlockLevel && userLevel >= flow.unlockLevel)
    .map(([key]) => key);
}

export default tutorialFlows;
