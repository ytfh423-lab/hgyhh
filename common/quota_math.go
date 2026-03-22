package common

import "math/big"

var (
	maxSafeQuotaBig = big.NewInt(MaxSafeQuota)
	minSafeQuotaBig = big.NewInt(-MaxSafeQuota)
)

func ClampQuotaInt64(v int64) int {
	if v > MaxSafeQuota {
		return int(MaxSafeQuota)
	}
	if v < -MaxSafeQuota {
		return -int(MaxSafeQuota)
	}
	return int(v)
}

func ClampQuotaFloat64(v float64) int {
	if v > float64(MaxSafeQuota) {
		return int(MaxSafeQuota)
	}
	if v < -float64(MaxSafeQuota) {
		return -int(MaxSafeQuota)
	}
	return int(v)
}

func ClampQuotaBigInt(v *big.Int) int {
	if v == nil {
		return 0
	}
	if v.Cmp(maxSafeQuotaBig) > 0 {
		return int(MaxSafeQuota)
	}
	if v.Cmp(minSafeQuotaBig) < 0 {
		return -int(MaxSafeQuota)
	}
	return int(v.Int64())
}

func SafeQuotaAdd(values ...int) int {
	var sum int64
	for _, value := range values {
		v := int64(value)
		if v > 0 && sum > MaxSafeQuota-v {
			return int(MaxSafeQuota)
		}
		if v < 0 && sum < -MaxSafeQuota-v {
			return -int(MaxSafeQuota)
		}
		sum += v
	}
	return int(sum)
}

func SafeQuotaMulDiv(base, multiplier, divisor int) int {
	if divisor == 0 || base == 0 || multiplier == 0 {
		return 0
	}
	result := new(big.Int).Mul(big.NewInt(int64(base)), big.NewInt(int64(multiplier)))
	result.Quo(result, big.NewInt(int64(divisor)))
	return ClampQuotaBigInt(result)
}
