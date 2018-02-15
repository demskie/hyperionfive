package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

// used colors
const (
	RED int8 = iota
	CYAN
	WHITE
	MAGENTA
)

var screenMutex sync.Mutex

type loggingMessage struct {
	color   int8
	theTime time.Time
	message string
}

const defaultTimeLayout string = "2006-01-02 15:04:05"

//t = time.Now().UTC().Format(defaultTimeLayout)
//t = time.Now()

func handleLogging() {
	for {
		var firstLog loggingMessage
		firstLog = <-GLOBAL.logInfoChan
		newLogs := map[int8]map[string][]time.Time{}
		newLogs[firstLog.color] = map[string][]time.Time{}
		newLogs[firstLog.color][firstLog.message] = []time.Time{firstLog.theTime}
		for i := 0; i < 10000; i++ {
			select {
			case nextLog, notClosed := <-GLOBAL.logInfoChan:
				if notClosed {
					_, colorExists := newLogs[nextLog.color]
					if colorExists == false {
						newLogs[nextLog.color] = map[string][]time.Time{}
					}
					_, messageExists := newLogs[nextLog.color][nextLog.message]
					if messageExists == false {
						newLogs[nextLog.color][nextLog.message] = []time.Time{}
					}
					newLogs[nextLog.color][nextLog.message] = append(
						newLogs[nextLog.color][nextLog.message], nextLog.theTime)
				} else {
					screenMutex.Lock()
					color.Red("GLOBAL.logInfoChan has closed unexpectedly")
					screenMutex.Unlock()
				}
			default:
				break
			}
		}

		screenMutex.Lock()
		for aColor := range newLogs {
			for aMsg, multipleTimes := range newLogs[aColor] {
				if len(multipleTimes) > 1 {
					earliestTime := multipleTimes[0]
					for _, thisTime := range multipleTimes {
						if thisTime.Before(earliestTime) {
							earliestTime = thisTime
						}
					}
					aMsg = (earliestTime.UTC().Format(defaultTimeLayout) + " " +
						aMsg + " (repeated " + fmt.Sprint(len(multipleTimes)) + " times)")
				} else {
					aMsg = multipleTimes[0].UTC().Format(defaultTimeLayout) + " " + aMsg
				}
				switch aColor {
				case RED:
					color.Red(aMsg)
				case CYAN:
					color.Cyan(aMsg)
				case WHITE:
					if len(GLOBAL.logInfoChan) < (oneMillion / 2) {
						color.White(aMsg)
					}
				case MAGENTA:
					if GLOBAL.debugModeBool &&
						len(GLOBAL.logInfoChan) < (oneMillion/2) {
						color.Magenta(aMsg)
					}
				}
			}
		}
		screenMutex.Unlock()
	}
}

func concatenateMessages(arrayOfStrings []string) (msg string) {
	for i, val := range arrayOfStrings {
		if i > 0 {
			msg += " "
		}
		msg += val
	}
	return
}

func checkError(err error, arrayOfStrings ...string) bool {
	if err == nil {
		return false
	}
	msg := concatenateMessages(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   RED,
		theTime: time.Now(),
		message: "/e " + msg + " (err) " + err.Error(),
	}
	return true
}

func logError(arrayOfStrings ...string) {
	msg := concatenateMessages(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   RED,
		theTime: time.Now(),
		message: "/e " + msg,
	}
}

func logDebug(arrayOfStrings ...string) {
	if GLOBAL.debugModeBool != true {
		return
	}
	msg := time.Now().UTC().Format(defaultTimeLayout)
	msg += " /d " + concatenateMessages(arrayOfStrings)
	screenMutex.Lock()
	color.Green(msg)
	screenMutex.Unlock()
}

func logInfo(arrayOfStrings ...string) {
	msg := concatenateMessages(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   WHITE,
		theTime: time.Now(),
		message: "/i " + msg,
	}
}

func logTelemetry(arrayOfStrings ...string) {
	msg := concatenateMessages(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   MAGENTA,
		theTime: time.Now(),
		message: "/t " + msg,
	}
}
