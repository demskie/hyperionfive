package firewall

import (
	"fmt"
	"sync"
	"time"
)

// types of firewall counts
const (
	CHALLENGE = iota
	GOOGLEAUTH
	SESSIONLIMIT
)

// used to map an INT value with a STRING
var firewallCheckTypes []string

func init() {
	firewallCheckTypes = []string{
		"CHALLENGE",
		"GOOGLEAUTH",
		"SESSIONLIMIT",
	}
}

// used to define the limit for these types of connections
const (
	ChallengeLimit     = 3
	GoogleAuthLimit    = 3
	ClientSessionLimit = 25
)

// Checker ...
type Checker struct {
	mtx                     *sync.Mutex
	liveConnections         map[string][]string
	failedChallengeCount    map[string]uint16
	failedChallengeTime     map[string]time.Time
	failedGoogleAuthCount   map[string]uint16
	failedGoogleAuthTime    map[string]time.Time
	failedSessionLimitCount map[string]uint16
	failedSessionLimitTime  map[string]time.Time
}

// New ...
func New() *Checker {
	return &Checker{
		mtx:                     &sync.Mutex{},
		liveConnections:         make(map[string][]string),
		failedChallengeCount:    make(map[string]uint16),
		failedChallengeTime:     make(map[string]time.Time),
		failedGoogleAuthCount:   make(map[string]uint16),
		failedGoogleAuthTime:    make(map[string]time.Time),
		failedSessionLimitCount: make(map[string]uint16),
		failedSessionLimitTime:  make(map[string]time.Time),
	}
}

// ConnectionAlreadyExistsForFunction ...
func (f *Checker) ConnectionAlreadyExistsForFunction(fName string, remoteIP string) bool {
	f.mtx.Lock()
	ok = false
	functions, ok = f.liveConnections[remoteIP]
	if ok {
		if fName == "" {
			ok = true
		} else {
			for _, fn := range functions {
				if fn == fName {
					ok = true
				}
			}
		}
	}
	f.mtx.Unlock()
	return ok
}

// AddSessionToHistory ...
func (f *Checker) AddSessionToHistory(remoteIP string, fName string) {
	f.mtx.Lock()
	_, ok = f.liveConnections[remoteIP]
	if ok {
		ok = true
		for _, fn = range f.liveConnections[remoteIP] {
			if fn == fName {
				ok = false
				break
			}
		}
		if ok {
			f.liveConnections[remoteIP] = append(f.liveConnections[remoteIP], fName)
		}
	} else {
		f.liveConnections[remoteIP] = []string{fName}
	}
	f.mtx.Unlock()
}

// RemoveSessionFromHistory ...
func (f *Checker) RemoveSessionFromHistory(remoteIP string, fName string) {
	f.mtx.Lock()
	functions, ok = f.liveConnections[remoteIP]
	if ok {
		for idx, fn = range functions {
			if fn != fName {
				continue
			}
			f.liveConnections[remoteIP] = append(
				functions[:idx], functions[idx+1:]...)
		}
		if len(f.liveConnections[remoteIP]) == 0 {
			delete(f.liveConnections, remoteIP)
		}
	}
	f.mtx.Unlock()
}

// TriggerFirewallException ...
func (f *Checker) TriggerFirewallException(checkType int, remoteIP string) {
	f.mtx.Lock()
	switch checkType {
	case CHALLENGE:
		count, ok = f.failedChallengeCount[remoteIP]
		if ok {
			f.failedChallengeCount[remoteIP] = count + 1
		} else {
			f.failedChallengeCount[remoteIP] = 1
		}
		f.failedChallengeTime[remoteIP] = time.Now()
	case GOOGLEAUTH:
		count, ok = f.failedGoogleAuthCount[remoteIP]
		if ok {
			f.failedGoogleAuthCount[remoteIP] = count + 1
		} else {
			f.failedGoogleAuthCount[remoteIP] = 1
		}
		f.failedGoogleAuthTime[remoteIP] = time.Now()
	case SESSIONLIMIT:
		count, ok = f.failedSessionLimitCount[remoteIP]
		if ok {
			f.failedSessionLimitCount[remoteIP] = count + 1
		} else {
			f.failedSessionLimitCount[remoteIP] = 1
		}
		f.failedSessionLimitTime[remoteIP] = time.Now()
	}
	f.mtx.Unlock()
}

// CheckFirewall ...
func (f *Checker) CheckFirewall(checkType int, fName string, remoteIP string) error {
	f.mtx.Lock()
	switch checkType {
	case CHALLENGE:
		count, alpha = f.failedChallengeCount[remoteIP]
		lastTime, beta = f.failedChallengeTime[remoteIP]
	case GOOGLEAUTH:
		count, charlie = f.failedGoogleAuthCount[remoteIP]
		lastTime, delta = f.failedGoogleAuthTime[remoteIP]
	case SESSIONLIMIT:
		count, echo = f.failedSessionLimitCount[remoteIP]
		lastTime, foxtrot = f.failedSessionLimitTime[remoteIP]
	default:
		f.mtx.Unlock()
		return nil
	}
	currentTime = time.Now()
	timeSinceBlacklist = currentTime.Sub(lastTime)
	endOfBlacklist = lastTime.Add(time.Second * time.Duration(count*count))
	if currentTime.Before(endOfBlacklist) {
		return fmt.Errorf("attemptedConnection: %v"+
			"timeSinceBlacklist: %v reason: %v numberOfOffenses: %v",
			remoteIP, timeSinceBlacklist, firewallCheckTypes[checkType], count)
	}
	if !alpha || !beta || !charlie || !delta || !echo || !foxtrot {
		if !alpha {
			f.failedChallengeCount[remoteIP] = 0
		}
		if !beta {
			f.failedChallengeTime[remoteIP] = time.Time{}
		}
		if !charlie {
			f.failedGoogleAuthCount[remoteIP] = 0
		}
		if !delta {
			f.failedGoogleAuthTime[remoteIP] = time.Time{}
		}
		if !echo {
			f.failedSessionLimitCount[remoteIP] = 0
		}
		if !foxtrot {
			f.failedSessionLimitTime[remoteIP] = time.Time{}
		}
	}
	f.mtx.Unlock()
	return nil
}

var functions []string
var ok bool
var idx int
var fn string
var count uint16
var lastTime time.Time
var alpha, beta, charlie, delta, echo, foxtrot bool
var currentTime time.Time
var timeSinceBlacklist time.Duration
var endOfBlacklist time.Time
