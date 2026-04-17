package model

import "testing"

// TgFarmRandomEvent 的结构是否成立，以及对关键字段的零值假设。
// 这里只做 compile-time 合法性的冒烟测试，真正的 DB 操作走集成测试。

func TestRandomEvent_ZeroValueChosenIdx(t *testing.T) {
	ev := TgFarmRandomEvent{}
	// 新建事件默认 ChosenIdx=0 不行，我们的 Pending 判定是 chosen_idx = -1
	// 因此 controller 创建时必须显式置 -1；本测试防御性检查结构体默认值
	if ev.ChosenIdx != 0 {
		t.Error("expected go zero-value 0 for int field")
	}
}
