package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/logging"

	"github.com/gorilla/websocket"
	oauth2 "google.golang.org/api/oauth2/v1"
)

// HandleGoogleAuth ...
func HandleGoogleAuth(fName string, conn *websocket.Conn,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) (userid string) {

	fName += " (googleAuth)"

	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	err := fw.CheckFirewall(firewall.GOOGLEAUTH, fName, remoteIP)
	if log.CheckError(err) {
		return userid
	}

	type incomingMessage struct {
		LoginType string `json:"login_type"`
		IDToken   string `json:"id_token"`
	}
	var inMsg incomingMessage
	err = conn.ReadJSON(&inMsg)
	if log.CheckError(err, fName, "error receiving message") {
		return userid
	}
	log.Dump(inMsg)

	successful := false
	var oauth2Service *oauth2.Service
	for i := 0; i < firewall.GoogleAuthLimit; i++ {
		var httpClient = &http.Client{}
		oauth2Service, err = oauth2.New(httpClient)
		if log.CheckError(err, fName, "error contacting google") {
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
	if log.CheckError(err, fName, "error parsing incoming google response") {
		return userid
	}

	if tokenInfo.EmailVerified != true {
		log.Error(fName, "tokenInfo.EmailVerified is not 'true'", remoteIP)
		fw.TriggerFirewallException(firewall.GOOGLEAUTH, remoteIP)
		return userid
	}
	if tokenInfo.Audience != args.GoogleAudience {
		log.Error(fName, fmt.Sprintf("tokenInfo.Audience (%s) does not match googleAudience (%s) remoteIP=%s",
			tokenInfo.Audience, args.GoogleAudience, remoteIP))
		fw.TriggerFirewallException(firewall.GOOGLEAUTH, remoteIP)
		return userid
	}

	if args.DebugModeBool {
		rawJSON, err := tokenInfo.MarshalJSON()
		log.CheckError(err, fName, "error MarshalJSONing google response")
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, rawJSON, "", "    ")
		log.CheckError(err, fName, "error jsonIndenting google response")
		log.Debug(fName, "received the following message from google:\n", string(prettyJSON.Bytes()))
	}

	return tokenInfo.UserId
}
