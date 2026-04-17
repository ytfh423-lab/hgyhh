package model

import "testing"

// TestSoilScore_Clamp 检查 SoilScore 边界不会溢出
func TestSoilScore_Clamp(t *testing.T) {
	cases := []struct {
		name string
		plot TgFarmPlot
		min  int
		max  int
	}{
		{"zero", TgFarmPlot{}, 0, 100},
		{"perfect", TgFarmPlot{SoilN: 100, SoilP: 100, SoilK: 100, SoilPH: 65, SoilOM: 100, SoilFatigue: 0}, 95, 100},
		{"ruined", TgFarmPlot{SoilN: 0, SoilP: 0, SoilK: 0, SoilPH: 45, SoilOM: 0, SoilFatigue: 100}, 0, 20},
	}
	for _, c := range cases {
		got := SoilScore(&c.plot)
		if got < c.min || got > c.max {
			t.Errorf("%s: score=%d expected in [%d,%d]", c.name, got, c.min, c.max)
		}
	}
}

// TestSoilYieldFactor_Tiers 检查分档点
func TestSoilYieldFactor_Tiers(t *testing.T) {
	tests := []struct {
		plot TgFarmPlot
		want float64
	}{
		{TgFarmPlot{SoilN: 100, SoilP: 100, SoilK: 100, SoilPH: 65, SoilOM: 100}, 1.30},
		{TgFarmPlot{SoilN: 80, SoilP: 80, SoilK: 80, SoilPH: 65, SoilOM: 60}, 1.15},
		{TgFarmPlot{SoilN: 60, SoilP: 60, SoilK: 60, SoilPH: 65, SoilOM: 40}, 1.00},
		{TgFarmPlot{SoilN: 30, SoilP: 30, SoilK: 30, SoilPH: 65, SoilOM: 20}, 0.85},
		{TgFarmPlot{SoilN: 0, SoilP: 0, SoilK: 0, SoilPH: 45, SoilOM: 0, SoilFatigue: 100}, 0.70},
	}
	for i, tt := range tests {
		got := SoilYieldFactor(&tt.plot)
		if got != tt.want {
			t.Errorf("case %d: got %v want %v", i, got, tt.want)
		}
	}
}

// TestClampInt 边界
func TestClampInt(t *testing.T) {
	if clampInt(-5, 0, 100) != 0 {
		t.Error("clamp low failed")
	}
	if clampInt(200, 0, 100) != 100 {
		t.Error("clamp high failed")
	}
	if clampInt(50, 0, 100) != 50 {
		t.Error("clamp mid failed")
	}
}
