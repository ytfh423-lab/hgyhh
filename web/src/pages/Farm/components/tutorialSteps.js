/**
 * 强制式交互教学步骤配置
 *
 * 步骤字段：
 * - id:              唯一标识
 * - flowKey:         所属教程流程 key
 * - featureKey:      对应功能 key（与后端 feature_key 一致）
 * - page:            该步骤所在页面 key
 * - stepKey:         步骤键名
 * - title:           标题
 * - content:         说明文案
 * - targetSelector:  要高亮的 DOM 元素选择器
 * - placement:       提示框位置 (top / bottom / left / right / center)
 * - actionType:      交互类型
 *     highlight-only   只高亮展示（说明步骤，允许手动下一步）
 *     navigate         需要切换到指定页面
 *     wait-action      等待真实操作完成（监听 tutorialEvents）
 * - actionEvent:     需要等待的事件名（actionType=wait-action 时必填）
 * - actionPayload:   事件验证负载条件（可选）
 * - navigateTo:      目标页面 key（actionType=navigate 时）
 * - allowManualNext: 是否允许手动点"下一步"（false=必须完成动作）
 * - lockScope:       教程期间锁定范围 ('full' / 'page' / null)
 * - required:        是否必须步骤
 * - sortOrder:       排序序号
 */

// ══════════════════════════════════════════════
//  基础教程流程 (farm_basic) — 首次进入，不可跳过
// ══════════════════════════════════════════════
const farmBasicSteps = [
  // ── 阶段1: 认识总览 ──
  {
    id: 'fb-1', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'welcome', title: '欢迎来到农场！',
    content: '这是你的专属农场。接下来我会手把手教你完成第一次种植、浇水、收获和出售——跟着做就行！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 1,
  },
  {
    id: 'fb-2', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'statusbar', title: '资源栏',
    content: '最上方是你的资源栏：💰金币、⭐等级、🌾地块数量、☁️天气。所有操作都会影响这些数值。',
    targetSelector: '.farm-statusbar', placement: 'bottom',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 2,
  },
  {
    id: 'fb-3', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'plot-grid', title: '你的农田',
    content: '这些是你的地块。空地可以种植作物。接下来我们去种下你的第一颗作物！',
    targetSelector: '.farm-plot-grid', placement: 'top',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 3,
  },

  // ── 阶段2: 首次种植教学 ──
  {
    id: 'fb-4', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'go-plant', title: '前往种植页',
    content: '点击左侧导航栏的「🌱 种植」进入种植页面。',
    targetSelector: '[data-tutorial="nav-plant"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'plant',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 4,
  },
  {
    id: 'fb-5', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'plant',
    stepKey: 'select-crop', title: '选择作物',
    content: '点击任意一种作物来选中它。新手推荐 🌾小麦 或 🥔土豆，价格便宜、见效快！',
    targetSelector: '[data-tutorial="plant-page"]', placement: 'center',
    actionType: 'wait-action', actionEvent: 'select-crop',
    allowManualNext: false, lockScope: 'page', required: true, sortOrder: 5,
  },
  {
    id: 'fb-6', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'plant',
    stepKey: 'do-plant', title: '种植到空地',
    content: '选好作物后，点击下方任意一块空地完成种植！',
    targetSelector: '[data-tutorial="plant-page"]', placement: 'center',
    actionType: 'wait-action', actionEvent: 'plant-crop',
    allowManualNext: false, lockScope: 'page', required: true, sortOrder: 6,
  },

  // ── 阶段3: 养成教学 ──
  {
    id: 'fb-7', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'plant',
    stepKey: 'back-overview-water', title: '返回总览',
    content: '种植成功！现在回到总览页面查看作物状态。',
    targetSelector: '[data-tutorial="nav-overview"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'overview',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 7,
  },
  {
    id: 'fb-8', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'view-growing', title: '查看作物状态',
    content: '看到了吗？你的作物正在生长中 🌱！进度条显示了当前的成长进度。作物需要浇水才能健康成长。',
    targetSelector: '.farm-plot-growing', placement: 'bottom',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'page', required: true, sortOrder: 8,
  },
  {
    id: 'fb-9', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'do-water', title: '给作物浇水',
    content: '点击快捷操作栏的「💧 浇水」按钮，给作物浇水加速生长！',
    targetSelector: '.farm-overview-actions', placement: 'bottom',
    actionType: 'wait-action', actionEvent: 'water-crop',
    allowManualNext: false, lockScope: 'page', required: true, sortOrder: 9,
  },

  // ── 阶段4: 收获教学 ──
  {
    id: 'fb-10', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'wait-harvest-intro', title: '等待成熟',
    content: '作物浇水成功！教学模式下我们加速了成长——现在点击「🌾 收获」按钮来收获你的作物！如果作物还未成熟，请先等待或再次浇水。',
    targetSelector: '.farm-overview-actions', placement: 'bottom',
    actionType: 'wait-action', actionEvent: 'harvest-crop',
    allowManualNext: false, lockScope: 'page', required: true, sortOrder: 10,
  },

  // ── 阶段5: 市场/出售教学 ──
  {
    id: 'fb-11', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'harvest-done', title: '收获成功！',
    content: '你已经成功收获了作物！收获物已存入仓库。接下来让我们去仓库看看并出售它。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 11,
  },
  {
    id: 'fb-12', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'go-warehouse', title: '前往仓库',
    content: '点击左侧导航栏的「📦 仓库」查看你的库存。',
    targetSelector: '[data-tutorial="nav-warehouse"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'warehouse',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 12,
  },
  {
    id: 'fb-13', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'warehouse',
    stepKey: 'do-sell', title: '出售作物',
    content: '点击仓库中任意作物右侧的「💰 出售」按钮，或点「全部出售」。完成你的第一笔交易！',
    targetSelector: '[data-tutorial="warehouse-items"]', placement: 'top',
    actionType: 'wait-action', actionEvent: 'sell-item',
    allowManualNext: false, lockScope: 'page', required: true, sortOrder: 13,
  },

  // ── 阶段6: 任务教学 ──
  {
    id: 'fb-14', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'warehouse',
    stepKey: 'sell-done', title: '出售成功！',
    content: '你赚到了第一笔金币 💰！接下来看看每日任务，完成任务可以获取额外奖励。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 14,
  },
  {
    id: 'fb-15', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'warehouse',
    stepKey: 'go-tasks', title: '前往任务',
    content: '点击左侧导航栏的「📝 任务」查看今日任务。',
    targetSelector: '[data-tutorial="nav-tasks"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'tasks',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 15,
  },
  {
    id: 'fb-16', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'tasks',
    stepKey: 'view-tasks', title: '每日任务',
    content: '这里列出了你的每日任务。完成对应操作后点击「领取」获得奖励。每天都会刷新新任务。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'page', required: true, sortOrder: 16,
  },

  // ── 完成 ──
  {
    id: 'fb-17', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'tasks',
    stepKey: 'finish', title: '基础教学完成！🎉',
    content: '恭喜！你已掌握农场的核心玩法：种植 → 浇水 → 收获 → 出售。随着等级提升会解锁更多功能，届时会有专属教学。点击右上角「📖」可随时回顾。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: null, required: true, sortOrder: 17,
  },
];

// ══════════════════════════════════════════════
//  树场教程 (treefarm) — 解锁时强制触发
// ══════════════════════════════════════════════
const treefarmSteps = [
  {
    id: 'tf-1', flowKey: 'treefarm', featureKey: 'treefarm', page: 'overview',
    stepKey: 'tree-unlock', title: '🌲 树场已解锁！',
    content: '恭喜！你解锁了树场系统。树场是长期投资——种下树木，定期收获果实或伐木获取大量资源。跟我来体验一次完整流程！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 1,
  },
  {
    id: 'tf-2', flowKey: 'treefarm', featureKey: 'treefarm', page: 'overview',
    stepKey: 'go-treefarm', title: '进入树场',
    content: '点击左侧导航栏的「🌲 树场」进入树场页面。',
    targetSelector: '[data-tutorial="nav-treefarm"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'treefarm',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 2,
  },
  {
    id: 'tf-3', flowKey: 'treefarm', featureKey: 'treefarm', page: 'treefarm',
    stepKey: 'tree-grid', title: '树位总览',
    content: '这里是你的林地。每个空树位都可以种植一棵树。点击空树位选择树种进行种植。',
    targetSelector: '.tree-farm-grid', placement: 'bottom',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'page', required: true, sortOrder: 3,
  },
  {
    id: 'tf-4', flowKey: 'treefarm', featureKey: 'treefarm', page: 'treefarm',
    stepKey: 'do-plant-tree', title: '种植一棵树',
    content: '点击一个空树位，选择树种，完成种植！新手推荐普通木材树。',
    targetSelector: '.tree-slot-status-0', placement: 'bottom',
    actionType: 'wait-action', actionEvent: 'plant-tree',
    allowManualNext: false, lockScope: 'page', required: true, sortOrder: 4,
  },
  {
    id: 'tf-5', flowKey: 'treefarm', featureKey: 'treefarm', page: 'treefarm',
    stepKey: 'tree-planted', title: '种植成功！🌱',
    content: '树木已种下！树木生长较慢，但产出价值高。你可以浇水加速生长。树木成熟后可以采集果实或直接伐木。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: null, required: true, sortOrder: 5,
  },
];

// ══════════════════════════════════════════════
//  市场教程 (market) — 解锁时强制触发
// ══════════════════════════════════════════════
const marketSteps = [
  {
    id: 'mk-1', flowKey: 'market', featureKey: 'market', page: 'overview',
    stepKey: 'market-unlock', title: '📈 市场已解锁！',
    content: '恭喜解锁市场系统！市场价格实时波动，学会观察趋势可以让你获得更高收益。来了解一下吧！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 1,
  },
  {
    id: 'mk-2', flowKey: 'market', featureKey: 'market', page: 'overview',
    stepKey: 'go-market', title: '进入市场',
    content: '点击左侧导航栏的「📈 市场」查看实时行情。',
    targetSelector: '[data-tutorial="nav-market"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'market',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 2,
  },
  {
    id: 'mk-3', flowKey: 'market', featureKey: 'market', page: 'market',
    stepKey: 'market-intro', title: '市场行情',
    content: '这里展示了所有商品的当前市场价格和趋势。绿色箭头表示价格上涨，红色表示下跌。在高价时出售可以赚更多！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: null, required: true, sortOrder: 3,
  },
];

// ══════════════════════════════════════════════
//  仓库教程 (warehouse) — 解锁时可选触发
// ══════════════════════════════════════════════
const warehouseSteps = [
  {
    id: 'wh-1', flowKey: 'warehouse', featureKey: 'warehouse', page: 'overview',
    stepKey: 'warehouse-unlock', title: '📦 仓库系统',
    content: '仓库可以暂存收获物，等市场价格高时再出售。注意物品有保质期——过期会损失！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: null, required: true, sortOrder: 1,
  },
];

// ══════════════════════════════════════════════
//  任务教程 (tasks) — 首次进入时触发
// ══════════════════════════════════════════════
const tasksSteps = [
  {
    id: 'tk-1', flowKey: 'tasks', featureKey: 'tasks', page: 'overview',
    stepKey: 'tasks-unlock', title: '📝 每日任务',
    content: '每天都会刷新新任务，完成任务可获得金币和经验奖励。记得每天登录领取！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: null, required: true, sortOrder: 1,
  },
];

// ══════════════════════════════════════════════
//  汇总导出
// ══════════════════════════════════════════════

/** 所有教程流程定义 */
export const tutorialFlows = {
  farm_basic: {
    featureKey: 'farm_basic',
    label: '基础教程',
    emoji: '🌾',
    steps: farmBasicSteps,
    forcedOnFirstEntry: true,  // 首次进入强制触发
  },
  treefarm: {
    featureKey: 'treefarm',
    label: '树场教程',
    emoji: '🌲',
    steps: treefarmSteps,
    unlockLevel: 5,
  },
  market: {
    featureKey: 'market',
    label: '市场教程',
    emoji: '📈',
    steps: marketSteps,
    unlockLevel: 2,
  },
  warehouse: {
    featureKey: 'warehouse',
    label: '仓库教程',
    emoji: '📦',
    steps: warehouseSteps,
    unlockLevel: 1,
  },
  tasks: {
    featureKey: 'tasks',
    label: '任务教程',
    emoji: '📝',
    steps: tasksSteps,
    unlockLevel: 1,
  },
};

/** 获取指定流程的步骤列表 */
export function getFlowSteps(flowKey) {
  return tutorialFlows[flowKey]?.steps || [];
}

/** 获取所有需要在特定等级触发的流程 key 列表 */
export function getUnlockableFlows(userLevel) {
  return Object.entries(tutorialFlows)
    .filter(([, flow]) => flow.unlockLevel && userLevel >= flow.unlockLevel)
    .map(([key]) => key);
}

export default tutorialFlows;
