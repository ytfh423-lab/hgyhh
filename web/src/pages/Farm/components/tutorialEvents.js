/**
 * tutorialEvents — 教程事件总线
 *
 * 用于在农场业务操作（种植/浇水/收获/出售等）和教程系统之间解耦通信。
 * 业务操作成功后 emit 事件，TutorialProvider 监听并推进教程步骤。
 *
 * 事件类型：
 *   action:plant-crop      种植作物成功
 *   action:water-crop      浇水成功
 *   action:fertilize-crop  施肥成功
 *   action:harvest-crop    收获成功
 *   action:harvest-store   收获到仓库成功
 *   action:sell-item       仓库出售成功
 *   action:sell-all        全部出售成功
 *   action:open-page       打开了某个页面 (payload: { page })
 *   action:click-plot      点击了地块 (payload: { plotIndex })
 *   action:select-crop     选择了作物 (payload: { cropKey })
 *   action:plant-tree      种树成功
 *   action:water-tree      给树浇水成功
 *   action:harvest-tree    采集树木成功
 *   action:chop-tree       伐木成功
 *   action:claim-task      领取任务奖励成功
 */

class TutorialEventBus {
  constructor() {
    this._listeners = {};
  }

  on(event, callback) {
    if (!this._listeners[event]) {
      this._listeners[event] = [];
    }
    this._listeners[event].push(callback);
    return () => this.off(event, callback);
  }

  off(event, callback) {
    if (!this._listeners[event]) return;
    this._listeners[event] = this._listeners[event].filter(cb => cb !== callback);
  }

  emit(event, payload) {
    if (!this._listeners[event]) return;
    this._listeners[event].forEach(cb => {
      try { cb(payload); } catch (e) { console.error('[TutorialEvents]', e); }
    });
  }

  // 便捷方法：监听所有 action:* 事件
  onAction(callback) {
    return this.on('action:*', callback);
  }

  emitAction(actionType, payload) {
    this.emit(`action:${actionType}`, payload);
    this.emit('action:*', { type: actionType, ...payload });
  }
}

const tutorialEvents = new TutorialEventBus();

export default tutorialEvents;
