package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"math/rand"
	"os"
	"strconv"
	"time"

	_ "net/http/pprof" //

	"github.com/fatih/color"
	"github.com/gorilla/websocket"

	cryptorand "crypto/rand"
)

func main() {
	const fName = "(main)"
	switch {
	case GLOBAL.enableFrontendBool:
		startFrontendMode() // frontend.go
	case GLOBAL.enableBackendBool:
		startBackendMode() // backend.go
	default:
		logError(fName, "no operating mode was specified")
	}
	suddenExit()
}

////////////////////////////////////////////////////////////
// the rest of this file contains generic functions
////////////////////////////////////////////////////////////

func suddenExit() {
	i := 0
	time.Sleep(time.Second)
	for len(GLOBAL.logInfoChan) != 0 {
		logError("(suddenExit) shutting down in", fmt.Sprint(15-i), "seconds")
		time.Sleep(time.Second)
		if i >= 15 {
			break
		}
		i++
	}
	t := time.Now().UTC().Format("2006-01-02 15:04:05")
	color.Red(t + " /e (suddenExit) program has finished executing")
	color.Unset()
	os.Exit(1)
}

func getLocalAddress() (tcpAddr net.TCPAddr) {
	if GLOBAL.localInterface != "" {
		iface, err := net.InterfaceByName(GLOBAL.localInterface)
		if checkError(err) == false {
			addrs, anotherError := iface.Addrs()
			if checkError(anotherError) == false {
				tcpAddr = net.TCPAddr{
					IP: addrs[0].(*net.IPNet).IP,
				}
			}
		}
	}
	if GLOBAL.localAddress != "" {
		tcpAddr = net.TCPAddr{
			IP: net.ParseIP(GLOBAL.localAddress),
		}
	} else {
		ifaces, err := net.Interfaces()
		if checkError(err) == false {
			for _, iface := range ifaces {
				addrs, err := iface.Addrs()
				if checkError(err) {
					continue
				}
				for _, addr := range addrs {
					if checkError(err) {
						continue
					}
					tcpAddr = net.TCPAddr{
						IP: addr.(*net.IPNet).IP,
					}
					return tcpAddr
				}
			}
		}
	}
	return tcpAddr
}

func getWebsocketDialer() (dialer *websocket.Dialer) {
	tcpAddr := getLocalAddress()
	netDialer := &net.Dialer{
		LocalAddr: &tcpAddr,
		Deadline:  time.Now().Add(time.Second * 30),
	}
	// logDebug(fmt.Sprintf("netDialer = %+v", netDialer))
	dialer = &websocket.Dialer{
		NetDial: netDialer.Dial,
	}
	// logDebug(fmt.Sprintf("*websocket.Dialer = %+v", dialer))
	return dialer
}

func absInt64(val int64) int64 {
	if val < 0 {
		val = -val
	}
	return val
}

func commaFormatNum(n interface{}) string {
	// BUG // starting to miss python
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

func getStartAndStopIndexForCore(coreNum, entireSliceLength int) (startIndex, stopIndex int) {
	if GLOBAL.numberOfCores > entireSliceLength {
		if coreNum < entireSliceLength {
			return coreNum, coreNum + 1
		}
		return 0, -1
	}
	stopIndex = int(entireSliceLength/GLOBAL.numberOfCores) * (coreNum + 1)
	startIndex = stopIndex - int(entireSliceLength/GLOBAL.numberOfCores)
	if coreNum == GLOBAL.numberOfCores-1 {
		stopIndex = entireSliceLength
	}
	return startIndex, stopIndex
}

func mulOverflows(a, b uint64) bool {
	if a <= 1 || b <= 1 {
		return false
	}
	c := a * b
	return c/b != a
}

const mostNegative = -(mostPositive + 1)
const mostPositive = 1<<63 - 1

func signedMulOverflows(a, b int64) bool {
	if a == 0 || b == 0 || a == 1 || b == 1 {
		return false
	}
	if a == mostNegative || b == mostNegative {
		return true
	}
	c := a * b
	return c/b != a
}

func appendStringIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}

func removeStringIfPresent(slice []string, s string) []string {
	for i, ele := range slice {
		if ele == s {
			slice[i] = slice[len(slice)-1]
			slice = slice[:len(slice)-1]
		}
	}
	return slice
}

func generateRandomUint64() uint64 {
	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	rnum := rand.New(seed())
	return rnum.Uint64()
}

func getCryptoRnum() *rand.Rand {
	seed := func() rand.Source {
		var randomness int64
		b := make([]byte, 8)
		cryptorand.Read(b)
		buf := bytes.NewBuffer(b)
		binary.Read(buf, binary.LittleEndian, &randomness)
		return rand.NewSource(randomness)
	}
	return rand.New(seed())
}

func getPosOrNegOne(rnum *rand.Rand) int {
	if rnum.Intn(2) == 0 {
		return 1
	}
	return -1
}

func absInt32(val int32) int32 {
	if val < 0 {
		return -val
	}
	return val
}

func text(stuff ...interface{}) (result string) {
	for i := range stuff {
		result += "%v"
		if i < len(stuff)-1 {
			result += " "
		}
	}
	return fmt.Sprintf(result, stuff...)
}

type weightedObj struct {
	Weight int
	Item   interface{}
}

func weightedChoice(choices []weightedObj, rnum *rand.Rand) (weightedObj, error) {
	max := 0
	for _, choice := range choices {
		max += choice.Weight
	}
	num := rnum.Intn(max)
	for _, choice := range choices {
		num -= choice.Weight
		if num <= 0 {
			return choice, nil
		}
	}
	err := errors.New("weightedChoice func is broken")
	return weightedObj{}, err
}

func getNumberString(num int) string {
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
