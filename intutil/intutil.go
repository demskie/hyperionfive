package intutil

import (
	"fmt"
	"strconv"
)

// GetStartStopForCore ...
func GetStartStopForCore(coreNum, totalCoreNum, entireSliceLength int) (startIndex, stopIndex int) {
	if totalCoreNum > entireSliceLength {
		if coreNum < entireSliceLength {
			return coreNum, coreNum + 1
		}
		return 0, -1
	}
	stopIndex = int(entireSliceLength/totalCoreNum) * (coreNum + 1)
	startIndex = stopIndex - int(entireSliceLength/totalCoreNum)
	if coreNum == totalCoreNum-1 {
		stopIndex = entireSliceLength
	}
	return startIndex, stopIndex
}

// CommaFormatNum ...
func CommaFormatNum(n interface{}) string {
	var val uint64
	switch n.(type) {
	case int:
		val = uint64(n.(int))
	case int8:
		val = uint64(n.(int8))
	case int16:
		val = uint64(n.(int16))
	case int32:
		val = uint64(n.(int32))
	case int64:
		val = uint64(n.(int64))
	case uint:
		val = uint64(n.(uint))
	case uint8:
		val = uint64(n.(uint8))
	case uint16:
		val = uint64(n.(uint16))
	case uint32:
		val = uint64(n.(uint32))
	case uint64:
		val = uint64(n.(uint64))
	case float32:
		val = uint64(n.(float32))
	case float64:
		val = uint64(n.(float64))
	default:
		return "NaN"
	}
	in := strconv.FormatUint(val, 10)
	out := make([]byte, len(in)+(len(in)-2+int(in[0]/'0'))/3)
	if in[0] == '-' {
		in, out[0] = in[1:], '-'
	}
	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}

// AbsInt32 ...
func AbsInt32(val int32) int32 {
	if val < 0 {
		return -val
	}
	return val
}

// AbsInt64 ...
func AbsInt64(val int64) int64 {
	if val < 0 {
		val = -val
	}
	return val
}

// MulOverflows ...
func MulOverflows(a, b uint64) bool {
	if a <= 1 || b <= 1 {
		return false
	}
	c := a * b
	return c/b != a
}

const mostNegative = -(mostPositive + 1)
const mostPositive = 1<<63 - 1

// SignedMulOverflows ...
func SignedMulOverflows(a, b int64) bool {
	if a == 0 || b == 0 || a == 1 || b == 1 {
		return false
	}
	if a == mostNegative || b == mostNegative {
		return true
	}
	c := a * b
	return c/b != a
}

// GetNumberString ...
func GetNumberString(num int) string {
	mappings := map[int]string{
		1:  "one",
		2:  "two",
		3:  "three",
		4:  "four",
		5:  "five",
		6:  "six",
		7:  "seven",
		8:  "eight",
		9:  "nine",
		10: "ten",
		11: "eleven",
		12: "twelve",
		13: "thirteen",
		14: "fourteen",
		15: "fifteen",
		16: "sixteen",
		17: "seventeen",
		18: "eighteen",
		19: "nineteen",
		20: "twenty",
		21: "twenty-one",
		22: "twenty-two",
		23: "twenty-three",
		24: "twenty-four",
		25: "twenty-five",
		26: "twenty-six",
		27: "twenty-seven",
		28: "twenty-eight",
		29: "twenty-nine",
		30: "thirty",
		31: "thirty-one",
		32: "thirty-two",
		33: "thirty-three",
		34: "thirty-four",
		35: "thirty-five",
		36: "thirty-six",
		37: "thirty-seven",
		38: "thirty-eight",
		39: "thirty-nine",
		40: "forty",
		41: "forty-one",
		42: "forty-two",
		43: "forty-three",
		44: "forty-four",
		45: "forty-five",
		46: "forty-six",
		47: "forty-seven",
		48: "forty-eight",
		49: "forty-nine",
		50: "fifty",
		51: "fifty-one",
		52: "fifty-two",
		53: "fifty-three",
		54: "fifty-four",
		55: "fifty-five",
		56: "fifty-six",
		57: "fifty-seven",
		58: "fifty-eight",
		59: "fifty-nine",
		60: "sixty",
		61: "sixty-one",
		62: "sixty-two",
		63: "sixty-three",
		64: "sixty-four",
	}
	result, ok := mappings[num]
	if !ok {
		return fmt.Sprint(num)
	}
	return result
}
