package main

import (
	"time"

	"github.com/fatih/color"
)

// used colors
const (
	RED int8 = iota
	CYAN
	WHITE
)

type loggingMessage struct {
	color   int8
	message string
}

const defaultTimeLayout string = "2006-01-02 15:04:05"

func handleLogging() {
	for {
		var firstLog loggingMessage
		firstLog = <-GLOBAL.logInfoChan
		newMessages := map[int8][]string{}
		newMessages[firstLog.color] = []string{firstLog.message}
		for i := 0; i < 10000; i++ {
			select {
			case nextLog, doesExist := <-GLOBAL.logInfoChan:
				if doesExist {
					_, ok := newMessages[nextLog.color]
					if ok {
						newMessages[nextLog.color] = append(
							newMessages[nextLog.color], nextLog.message)
					} else {
						newMessages[nextLog.color] = []string{nextLog.message}
					}
				}
			default:
				break
			}
		}
		GLOBAL.loggingMutex.Lock()
		for colorcode, messages := range newMessages {
			switch colorcode {
			case RED:
				for _, msg := range messages {
					color.Red(msg)
				}
			case CYAN:
				for _, msg := range messages {
					color.Cyan(msg)
				}
			case WHITE:
				for _, msg := range messages {
					color.White(msg)
				}
			}
		}
		GLOBAL.loggingMutex.Unlock()
	}
}

func prepareMessage(arrayOfStrings []string) (t, msg string) {
	for i, val := range arrayOfStrings {
		if i > 0 {
			msg += " "
		}
		msg += val
	}
	t = time.Now().UTC().Format(defaultTimeLayout)
	return
}

func logInfo(arrayOfStrings ...string) {
	t, msg := prepareMessage(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   WHITE,
		message: t + " /i " + msg,
	}
}

func logDebug(arrayOfStrings ...string) {
	if GLOBAL.debugModeBool != true {
		return
	}
	t, msg := prepareMessage(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   CYAN,
		message: t + " /d " + msg,
	}
}

func checkError(err error, arrayOfStrings ...string) bool {
	if err == nil {
		return false
	}
	t, msg := prepareMessage(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   RED,
		message: t + " /e " + msg,
	}
	return true
}

func logError(arrayOfStrings ...string) {
	time, msg := prepareMessage(arrayOfStrings)
	GLOBAL.logInfoChan <- loggingMessage{
		color:   RED,
		message: time + " /e " + msg,
	}
}
