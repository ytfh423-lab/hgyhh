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
  // ── 欢迎 ──
  {
    id: 'fb-1', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'welcome', title: '欢迎来到农场！',
    content: '这是你的专属农场 🌾 接下来我会带你快速了解每个功能区域——不需要操作，看完就能上手！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 1,
  },

  // ── 总览页介绍 ──
  {
    id: 'fb-2', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'statusbar', title: '💰 资源栏',
    content: '最上方是你的资源栏：金币余额、等级经验、地块数量和当前天气。所有操作都会影响这些数值。',
    targetSelector: '.farm-statusbar', placement: 'bottom',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 2,
  },
  {
    id: 'fb-3', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'plot-grid', title: '🌱 农田地块',
    content: '这里展示你的所有地块。空地可以种植作物，种下后会显示生长进度。作物成熟后可以收获。',
    targetSelector: '.farm-plot-grid', placement: 'top',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 3,
  },
  {
    id: 'fb-4', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'quick-actions', title: '⚡ 快捷操作',
    content: '这里有「全部浇水」和「全部施肥」按钮，可以一键照料所有作物。浇水保持生长，施肥加速成熟。',
    targetSelector: '.farm-overview-actions', placement: 'bottom',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 4,
  },

  // ── 种植页介绍 ──
  {
    id: 'fb-5', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'overview',
    stepKey: 'go-plant', title: '前往种植页',
    content: '点击左侧「🌱 种植」查看可种植的作物。',
    targetSelector: '[data-tutorial="nav-plant"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'plant',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 5,
  },
  {
    id: 'fb-6', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'plant',
    stepKey: 'plant-intro', title: '🌱 种植页',
    content: '这里展示所有可种植的作物，包括种子价格、生长时间和收益。选中作物后点击空地即可种植。新手推荐 🌾小麦 或 🥔土豆——便宜又快！',
    targetSelector: '[data-tutorial="plant-page"]', placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 6,
  },

  // ── 仓库页介绍 ──
  {
    id: 'fb-7', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'plant',
    stepKey: 'go-warehouse', title: '前往仓库',
    content: '点击左侧「📦 仓库」查看你的库存。',
    targetSelector: '[data-tutorial="nav-warehouse"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'warehouse',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 7,
  },
  {
    id: 'fb-8', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'warehouse',
    stepKey: 'warehouse-intro', title: '📦 仓库',
    content: '收获的作物会存入仓库。你可以单独出售或全部出售。注意季节影响价格——应季便宜、反季贵，学会囤货等好时机出手！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 8,
  },

  // ── 市场页介绍 ──
  {
    id: 'fb-9', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'warehouse',
    stepKey: 'go-market', title: '前往市场',
    content: '点击左侧「� 市场」查看当前行情。',
    targetSelector: '[data-tutorial="nav-market"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'market',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 9,
  },
  {
    id: 'fb-10', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'market',
    stepKey: 'market-intro', title: '📈 市场行情',
    content: '市场价格实时波动！绿色↑表示涨价，红色↓表示跌价。在高价时出售作物可以赚更多金币。点击商品可查看详细走势。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 10,
  },

  // ── 商店页介绍 ──
  {
    id: 'fb-11', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'market',
    stepKey: 'go-shop', title: '前往商店',
    content: '点击左侧「🏪 商店」看看有什么好东西。',
    targetSelector: '[data-tutorial="nav-shop"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'shop',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 11,
  },
  {
    id: 'fb-12', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'shop',
    stepKey: 'shop-intro', title: '🏪 商店',
    content: '商店出售各种实用道具：化肥加速生长、农药治虫、新地块扩展农田。随等级提升会解锁更多商品。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 12,
  },

  // ── 任务页介绍 ──
  {
    id: 'fb-13', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'shop',
    stepKey: 'go-tasks', title: '前往任务',
    content: '点击左侧「📝 任务」查看每日任务。',
    targetSelector: '[data-tutorial="nav-tasks"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'tasks',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 13,
  },
  {
    id: 'fb-14', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'tasks',
    stepKey: 'tasks-intro', title: '📝 每日任务',
    content: '每天会刷新新任务，完成对应操作后点击「领取」获得金币和经验奖励。坚持做任务是升级的好方法！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 14,
  },

  // ── 完成 ──
  {
    id: 'fb-15', flowKey: 'farm_basic', featureKey: 'farm_basic', page: 'tasks',
    stepKey: 'finish', title: '导览完成！🎉',
    content: '恭喜！你已了解农场的核心功能。基本流程：种植 → 浇水 → 收获 → 出售。随着等级提升会解锁牧场、钓鱼、加工坊等更多玩法。现在去种下你的第一颗作物吧！',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: null, required: true, sortOrder: 15,
  },
];

// ══════════════════════════════════════════════
//  树场教程 (treefarm) — 解锁时触发，可跳过
// ══════════════════════════════════════════════
const treefarmSteps = [
  {
    id: 'tf-1', flowKey: 'treefarm', featureKey: 'treefarm', page: 'overview',
    stepKey: 'tree-unlock', title: '🌲 树场已解锁！',
    content: '恭喜！你解锁了树场系统。树场是长期投资——种下树木，定期收获果实或伐木获取大量资源。',
    targetSelector: null, placement: 'center',
    actionType: 'highlight-only', allowManualNext: true, lockScope: 'full', required: true, sortOrder: 1,
  },
  {
    id: 'tf-2', flowKey: 'treefarm', featureKey: 'treefarm', page: 'overview',
    stepKey: 'go-treefarm', title: '进入树场',
    content: '点击左侧「🌲 树场」进入树场页面看看。',
    targetSelector: '[data-tutorial="nav-treefarm"]', placement: 'right',
    actionType: 'navigate', navigateTo: 'treefarm',
    allowManualNext: false, lockScope: 'full', required: true, sortOrder: 2,
  },
  {
    id: 'tf-3', flowKey: 'treefarm', featureKey: 'treefarm', page: 'treefarm',
    stepKey: 'tree-grid', title: '🌲 林地总览',
    content: '这里是你的林地。每个空树位可以种一棵树，点击空树位选择树种即可种植。树木生长较慢但产出价值高，可以浇水加速。新手推荐普通木材树。',
    targetSelector: '.tree-farm-grid', placement: 'bottom',
    actionType: 'highlight-only', allowManualNext: true, lockScope: null, required: true, sortOrder: 3,
  },
];

// ══════════════════════════════════════════════
//  市场教程 (market) — 解锁时触发，可跳过
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
//  仓库教程 (warehouse) — 解锁时触发，可跳过
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
//  任务教程 (tasks) — 解锁时触发，可跳过
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
