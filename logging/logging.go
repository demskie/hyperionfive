package logging

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// used colors
const (
	RED int8 = iota
	CYAN
	WHITE
	MAGENTA
)

type loggingMessage struct {
	color   int8
	theTime time.Time
	message string
}

// Logger ...
type Logger struct {
	mtx       *sync.Mutex
	debugMode bool
	logChan   chan loggingMessage
}

const oneMillion int = 1000000

// New ...
func New() *Logger {
	log := &Logger{
		mtx:       &sync.Mutex{},
		debugMode: false,
		logChan:   make(chan loggingMessage, oneMillion),
	}
	log.startLogging()
	log.Info("Logger has initialized")
	return log
}

// EnableDebugMode ...
func (l *Logger) EnableDebugMode() {
	l.debugMode = true
}

const defaultTimeLayout string = "2006-01-02 15:04:05"

func (l *Logger) startLogging() {
	var i int
	var aColor int8
	var aMsg string
	var earliestTime, thisTime time.Time
	var multipleTimes []time.Time
	var firstLog, nextLog loggingMessage
	var newLogs map[int8]map[string][]time.Time
	var notClosed, colorExists, messageExists bool

	go func() {
		for {
			firstLog = <-l.logChan
			newLogs = map[int8]map[string][]time.Time{}
			newLogs[firstLog.color] = map[string][]time.Time{}
			newLogs[firstLog.color][firstLog.message] = []time.Time{firstLog.theTime}
			for i = 0; i < 100000; i++ {
				select {
				case nextLog, notClosed = <-l.logChan:
					if notClosed {
						_, colorExists = newLogs[nextLog.color]
						if colorExists == false {
							newLogs[nextLog.color] = map[string][]time.Time{}
						}
						_, messageExists = newLogs[nextLog.color][nextLog.message]
						if messageExists == false {
							newLogs[nextLog.color][nextLog.message] = []time.Time{}
						}
						newLogs[nextLog.color][nextLog.message] = append(
							newLogs[nextLog.color][nextLog.message], nextLog.theTime)
					} else {
						panic("*Logger.logChan has closed unexpectedly")
					}
				default:
					break
				}
			}
			l.mtx.Lock()
			for aColor = range newLogs {
				for aMsg, multipleTimes = range newLogs[aColor] {
					if len(multipleTimes) > 1 {
						earliestTime = multipleTimes[0]
						for _, thisTime = range multipleTimes {
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
						fmt.Printf("\033[1;31m%s\033[0m", aMsg)
					case CYAN:
						fmt.Printf("\033[1;36m%s\033[0m", aMsg)
					case WHITE:
						fmt.Printf("\033[1;37m%s\033[0m", aMsg)
					case MAGENTA:
						if l.debugMode {
							fmt.Printf("\033[1;35m%s\033[0m", aMsg)
						}
					}
				}
			}
			l.mtx.Unlock()
		}
	}()
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

// CheckError ...
func (l *Logger) CheckError(err error, arrayOfStrings ...string) bool {
	if err == nil {
		return false
	}
	msg := concatenateMessages(arrayOfStrings)
	l.logChan <- loggingMessage{
		color:   RED,
		theTime: time.Now(),
		message: "/e " + msg + " (err) " + err.Error(),
	}
	return true
}

// Error ...
func (l *Logger) Error(arrayOfStrings ...string) {
	msg := concatenateMessages(arrayOfStrings)
	l.logChan <- loggingMessage{
		color:   RED,
		theTime: time.Now(),
		message: "/e " + msg,
	}
}

// Debug ...
func (l *Logger) Debug(arrayOfStrings ...string) {
	if l.debugMode == false {
		return
	}
	msg := time.Now().UTC().Format(defaultTimeLayout)
	msg += " /d " + concatenateMessages(arrayOfStrings)
	l.mtx.Lock()
	fmt.Printf("\033[1;32m%s\033[0m", msg)
	l.mtx.Unlock()
}

// Info ...
func (l *Logger) Info(arrayOfStrings ...string) {
	msg := concatenateMessages(arrayOfStrings)
	l.logChan <- loggingMessage{
		color:   WHITE,
		theTime: time.Now(),
		message: "/i " + msg,
	}
}

// Telemetry ...
func (l *Logger) Telemetry(arrayOfStrings ...string) {
	msg := concatenateMessages(arrayOfStrings)
	l.logChan <- loggingMessage{
		color:   MAGENTA,
		theTime: time.Now(),
		message: "/t " + msg,
	}
}

// SuddenExit ...
func (l *Logger) SuddenExit() {
	i := 0
	time.Sleep(time.Second)
	for len(l.logChan) != 0 {
		l.Error("(suddenExit) shutting down in", fmt.Sprint(15-i), "seconds")
		time.Sleep(time.Second)
		if i >= 15 {
			break
		}
		i++
	}
	t := time.Now().UTC().Format(defaultTimeLayout)
	l.mtx.Lock()
	fmt.Printf("\033[1;31m%s\033[0m /e (suddenExit) program has finished executing", t)
	os.Exit(1)
}
