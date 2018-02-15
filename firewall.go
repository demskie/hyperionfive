package main

import (
	"fmt"
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

func checkForExistingConnection(fName string, remoteIP string) bool {
	GLOBAL.clientBlacklist.RLock()
	result := false
	functions, ok := GLOBAL.clientBlacklist.LiveConnections[remoteIP]
	if ok {
		if fName == "" {
			result = true
		} else {
			for _, fn := range functions {
				if fn == fName {
					result = true
				}
			}
		}
	}
	GLOBAL.clientBlacklist.RUnlock()
	return result
}

func addSessionToHistory(remoteIP string, fName string) {
	logDebug("adding session", remoteIP, "spawned by", fName)
	GLOBAL.clientBlacklist.Lock()
	_, ok := GLOBAL.clientBlacklist.LiveConnections[remoteIP]
	if ok {
		GLOBAL.clientBlacklist.LiveConnections[remoteIP] = appendStringIfMissing(
			GLOBAL.clientBlacklist.LiveConnections[remoteIP], fName)
	} else {
		GLOBAL.clientBlacklist.LiveConnections[remoteIP] = []string{fName}
	}
	GLOBAL.clientBlacklist.Unlock()
}

// BUG // THIS DOES NOT WORK AT ALL
func removeSessionFromHistory(remoteIP string, fName string) {
	logDebug("removing session", remoteIP, "spawned by", fName)
	GLOBAL.clientBlacklist.Lock()
	functions, ok := GLOBAL.clientBlacklist.LiveConnections[remoteIP]
	if ok {
		for idx, fn := range functions {
			if fn != fName {
				continue
			}
			GLOBAL.clientBlacklist.LiveConnections[remoteIP] = append(
				functions[:idx], functions[idx+1:]...)
		}
		if len(GLOBAL.clientBlacklist.LiveConnections[remoteIP]) == 0 {
			delete(GLOBAL.clientBlacklist.LiveConnections, remoteIP)
		}
	}
	GLOBAL.clientBlacklist.Unlock()
}

func incrementFirewallCount(check int, remoteIP string) {
	GLOBAL.clientBlacklist.Lock()
	switch check {
	case CHALLENGE:
		val, ok := GLOBAL.clientBlacklist.FailedChallengeCount[remoteIP]
		if ok {
			GLOBAL.clientBlacklist.FailedChallengeCount[remoteIP] = val + 1
		} else {
			GLOBAL.clientBlacklist.FailedChallengeCount[remoteIP] = 1
		}
	case GOOGLEAUTH:
		val, ok := GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIP]
		if ok {
			GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIP] = val + 1
		} else {
			GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIP] = 1
		}
	case SESSIONLIMIT:
		val, ok := GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIP]
		if ok {
			GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIP] = val + 1
		} else {
			GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIP] = 1
		}
	default:
		logError("incrementFirewallCount was given an invalid check value of",
			fmt.Sprintf("%v", check))
	}
	GLOBAL.clientBlacklist.Unlock()
}

func decrementFirewallCount(check int, remoteIP string) {
	GLOBAL.clientBlacklist.Lock()
	switch check {
	case CHALLENGE:
		GLOBAL.clientBlacklist.FailedChallengeCount[remoteIP]--
	case GOOGLEAUTH:
		GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIP]--
	case SESSIONLIMIT:
		GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIP]--
	default:
		logError("decrementFirewallCount was given an invalid check value of",
			fmt.Sprintf("%v", check))
	}
	GLOBAL.clientBlacklist.Unlock()
}

func triggerFirewallException(check int, remoteIP string) {
	switch check {
	case CHALLENGE:
		incrementFirewallCount(check, remoteIP)
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.FailedChallengeTime[remoteIP] = time.Now()
		GLOBAL.clientBlacklist.Unlock()
	case GOOGLEAUTH:
		incrementFirewallCount(check, remoteIP)
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.FailedGoogleAuthTime[remoteIP] = time.Now()
		GLOBAL.clientBlacklist.Unlock()
	case SESSIONLIMIT:
		incrementFirewallCount(check, remoteIP)
		GLOBAL.clientBlacklist.Lock()
		GLOBAL.clientBlacklist.FailedSessionLimitTime[remoteIP] = time.Now()
		GLOBAL.clientBlacklist.Unlock()
	default:
		logError("triggerFirewallException was given an invalid check value of",
			fmt.Sprintf("%v", check))
	}
}

func checkFirewall(check int, fName string, remoteIP string) bool {
	var count uint16
	var lastTime time.Time
	var alpha, beta, charlie, delta, echo, foxtrot bool
	GLOBAL.clientBlacklist.RLock()
	switch check {
	case CHALLENGE:
		count, alpha = GLOBAL.clientBlacklist.FailedChallengeCount[remoteIP]
		lastTime, beta = GLOBAL.clientBlacklist.FailedChallengeTime[remoteIP]
	case GOOGLEAUTH:
		count, charlie = GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIP]
		lastTime, delta = GLOBAL.clientBlacklist.FailedGoogleAuthTime[remoteIP]
	case SESSIONLIMIT:
		count, echo = GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIP]
		lastTime, foxtrot = GLOBAL.clientBlacklist.FailedSessionLimitTime[remoteIP]
	default:
		GLOBAL.clientBlacklist.RUnlock()
		logError(fName, "gave checkFirewall an invalid 'check' parameter")
		return true // fail open
	}
	GLOBAL.clientBlacklist.RUnlock()
	currentTime := time.Now()
	timeSinceBlacklist := currentTime.Sub(lastTime)
	endOfBlacklist := lastTime.Add(time.Second * time.Duration(count*count))
	if currentTime.Before(endOfBlacklist) {
		logError(fName, "blacklisted client attempting to connect:", remoteIP,
			"\n\tblacklistedFor:", fmt.Sprintf("%v", timeSinceBlacklist),
			"\n\treason:", firewallCheckTypes[check],
			"\n\tnumberOfOffenses:", fmt.Sprintf("%v", count))
		return false
	}
	if !alpha || !beta || !charlie || !delta || !echo || !foxtrot {
		GLOBAL.clientBlacklist.Lock()
		if !alpha {
			GLOBAL.clientBlacklist.FailedChallengeCount[remoteIP] = 0
		}
		if !beta {
			GLOBAL.clientBlacklist.FailedChallengeTime[remoteIP] = time.Time{}
		}
		if !charlie {
			GLOBAL.clientBlacklist.FailedGoogleAuthCount[remoteIP] = 0
		}
		if !delta {
			GLOBAL.clientBlacklist.FailedGoogleAuthTime[remoteIP] = time.Time{}
		}
		if !echo {
			GLOBAL.clientBlacklist.FailedSessionLimitCount[remoteIP] = 0
		}
		if !foxtrot {
			GLOBAL.clientBlacklist.FailedSessionLimitTime[remoteIP] = time.Time{}
		}
		GLOBAL.clientBlacklist.Unlock()
	}
	return true
}
