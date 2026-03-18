/**
 * tutorialEvents — 教程事件总线 v2
 *
 * 所有事件携带 { success: boolean, data?: any } 结构。
 * TutorialProvider 只在 success === true 时推进步骤。
 *
 * 事件名约定：  action:<动作名>
 *   plant-crop / water-crop / fertilize-crop / harvest-crop
 *   harvest-store / sell-item / sell-all
 *   select-crop / plant-tree / water-tree / harvest-tree / chop-tree
 *   claim-task
 */

class TutorialEventBus {
  constructor() {
    this._listeners = {};
  }

  on(event, callback) {
    if (!this._listeners[event]) this._listeners[event] = [];
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

  /** 业务成功时调用 */
  emitSuccess(actionType, data) {
    const payload = { success: true, action: actionType, ...(data || {}) };
    this.emit(`action:${actionType}`, payload);
    this.emit('action:*', payload);
  }

  /** 业务失败时调用（可选） */
  emitFail(actionType, data) {
    const payload = { success: false, action: actionType, ...(data || {}) };
    this.emit(`action:${actionType}`, payload);
  }

  /** 兼容旧接口 */
  emitAction(actionType, data) {
    this.emitSuccess(actionType, data);
  }
}

const tutorialEvents = new TutorialEventBus();

export default tutorialEvents;
