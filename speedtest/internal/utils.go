package internal

import (
	"crypto/rand"
	"fmt"
	"math"
)

func GenerateUUID() (string, error) {
	randUUID := make([]byte, 16)
	_, err := rand.Read(randUUID)
	if err != nil {
		return "", err
	}
	randUUID[8] = randUUID[8]&^0xc0 | 0x80
	randUUID[6] = randUUID[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", randUUID[0:4], randUUID[4:6], randUUID[6:8], randUUID[8:10], randUUID[10:]), nil
}

// calcMAFilter Median-Averaging Filter
func _(list []int64) float64 {
	if len(list) == 0 {
		return 0
	}
	var sum int64 = 0
	n := len(list)
	if n == 0 {
		return 0
	}
	length := len(list)
	for i := 0; i < length-1; i++ {
		for j := i + 1; j < length; j++ {
			if list[i] > list[j] {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	for i := 1; i < n-1; i++ {
		sum += list[i]
	}
	return float64(sum) / float64(n-2)
}

func pautaFilter(vector []int64) []int64 {
	DBG().Println("Per capture unit")
	DBG().Printf("Raw Sequence len: %d\n", len(vector))
	DBG().Printf("Raw Sequence: %v\n", vector)
	if len(vector) == 0 {
		return vector
	}
	mean, _, std, _, _ := sampleVariance(vector)
	var retVec []int64
	for _, value := range vector {
		if math.Abs(float64(value-mean)) < float64(3*std) {
			retVec = append(retVec, value)
		}
	}
	DBG().Printf("Raw average: %dByte\n", mean)
	DBG().Printf("Pauta Sequence len: %d\n", len(retVec))
	DBG().Printf("Pauta Sequence: %v\n", retVec)
	return retVec
}

// sampleVariance sample Variance
func sampleVariance(vector []int64) (mean, variance, stdDev, min, max int64) {
	if len(vector) == 0 {
		return 0, 0, 0, 0, 0
	}
	var sumNum, accumulate int64
	min = math.MaxInt64
	max = math.MinInt64
	for _, value := range vector {
		sumNum += value
		if min > value {
			min = value
		}
		if max < value {
			max = value
		}
	}
	mean = sumNum / int64(len(vector))
	for _, value := range vector {
		accumulate += (value - mean) * (value - mean)
	}
	variance = accumulate / int64(len(vector)-1) // Bessel's correction
	stdDev = int64(math.Sqrt(float64(variance)))
	return
}
