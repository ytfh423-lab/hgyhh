package setting

// 请求风控配置
var (
	// 风控总开关
	RequestRiskControlEnabled = false

	// 短时高频并发控制：在 BurstWindow 秒内最多允许 BurstLimit 次请求（按 token 维度）
	RequestRiskControlBurstLimit  = 20  // 默认：10秒内最多20次
	RequestRiskControlBurstWindow = 10  // 单位：秒

	// Token 吞吐量控制：在 TokenWindow 分钟内最多消耗 TokenThreshold 个 token（按用户维度）
	RequestRiskControlTokenThreshold = 500000 // 默认：50万 token
	RequestRiskControlTokenWindow    = 5      // 单位：分钟
)
