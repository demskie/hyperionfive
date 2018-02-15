package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	oauth2 "google.golang.org/api/oauth2/v1"
)

func handleGoogleAuth(fName string, conn *websocket.Conn) (userid string) {
	fName += " (googleAuth)"

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	if checkFirewall(GOOGLEAUTH, fName, remoteIP) != true {
		return userid
	}

	type incomingMessage struct {
		LoginType string `json:"login_type"`
		IDToken   string `json:"id_token"`
	}
	var inMsg incomingMessage
	err := conn.ReadJSON(&inMsg)
	if checkError(err, fName, "error receiving message") {
		return userid
	}
	logDebug(fName, "received the following message\n", fmt.Sprintf("%+v", inMsg))

	successful := false
	var oauth2Service *oauth2.Service
	for i := 0; i < constGoogleAuth; i++ {
		var httpClient = &http.Client{}
		oauth2Service, err = oauth2.New(httpClient)
		if checkError(err, fName, "error contacting google") {
			continue
		} else {
			successful = true
			break
		}
	}
	if successful == false {
		return userid
	}

	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(inMsg.IDToken)
	tokenInfo, err := tokenInfoCall.Do()
	if checkError(err, fName, "error parsing incoming google response") {
		return userid
	}

	if tokenInfo.VerifiedEmail != true {
		logError(fName, "tokenInfo.VerifiedEmail is not 'true'", remoteIP)
		triggerFirewallException(GOOGLEAUTH, remoteIP)
		return userid
	}
	if tokenInfo.Audience != GLOBAL.googleAudience {
		aMessage := fmt.Sprintf("tokenInfo.Audience (%s) does not match googleAudience (%s) remoteIP=%s",
			tokenInfo.Audience, GLOBAL.googleAudience, remoteIP)
		logError(fName, aMessage)
		triggerFirewallException(GOOGLEAUTH, remoteIP)
		return userid
	}

	if GLOBAL.debugModeBool == true {
		rawJSON, err := tokenInfo.MarshalJSON()
		checkError(err, fName, "error MarshalJSONing google response")
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, rawJSON, "", "    ")
		checkError(err, fName, "error jsonIndenting google response")
		logDebug(fName, "received the following message from google:\n",
			string(prettyJSON.Bytes()))
	}

	userid = tokenInfo.UserId
	return userid
}
