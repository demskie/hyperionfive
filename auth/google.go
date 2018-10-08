package auth

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/alecthomas/units"
	"github.com/demskie/hyperionfive/arguments"
	"github.com/demskie/hyperionfive/clientmessage"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/logging"
	"github.com/gogo/protobuf/proto"

	"github.com/gorilla/websocket"
	oauth2 "google.golang.org/api/oauth2/v2"
)

func checkWithGoogle(fName string, idToken string, remoteIP string,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) (string, error) {
	var oauth2Service *oauth2.Service
	var httpClient = &http.Client{}
	oauth2Service, err := oauth2.New(httpClient)
	if log.CheckError(err, fName, "(code 6237F5)") {
		return "", errors.New("Unable to complete request (code 6237F5)")
	}
	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(idToken)
	tokenInfo, err := tokenInfoCall.Do()
	if log.CheckError(err, fName, "(code C7A568)") {
		return "", errors.New("Unable to complete request (code C7A568)")
	}
	if tokenInfo.Audience != args.GoogleAudience {
		log.Error(fName, fmt.Sprintf("googleAudience (%s) != (%s) remoteIP=%s",
			tokenInfo.Audience, args.GoogleAudience, remoteIP))
		fw.TriggerFirewallException(firewall.GOOGLEAUTH, remoteIP)
		return "", errors.New("Unable to complete request (code 3B8111)")
	}
	return tokenInfo.UserId, nil
}

// HandleGoogleAuth ...
func HandleGoogleAuth(fName string, conn *websocket.Conn, preauthUserIDs map[string]time.Time,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) (userid string, err error) {
	fName += " (googleAuth)"
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	err = fw.CheckFirewall(firewall.GOOGLEAUTH, fName, remoteIP)
	if log.CheckError(err, fName, "(code 708BD4)") {
		return userid, errors.New("Unable to complete request (code 708BD4)")
	}
	conn.SetReadLimit(128 * int64(units.Kilobyte))
	msgType, b, err := conn.ReadMessage()
	if log.CheckError(err, fName, "(code 83FF64)") {
		return userid, errors.New("Unable to complete request (code 83FF64)")
	}
	if msgType != websocket.BinaryMessage {
		return userid, errors.New("Unable to complete request (code 61F991)")
	}
	var inMsg clientmessage.Request
	proto.Unmarshal(b, &inMsg)
	if log.CheckError(err, fName, "(code 88473A)") {
		return userid, errors.New("Unable to complete request (code 88473A)")
	}
	log.Dump(inMsg)
	if inMsg.GoogleAuth == nil || inMsg.GoogleAuth.UserID == "" || inMsg.GoogleAuth.IdToken == "" {
		return userid, fmt.Errorf("Unable to complete request (code B94319)")
	}
	_, exists := preauthUserIDs[inMsg.GoogleAuth.UserID]
	if exists {
		return inMsg.GoogleAuth.UserID, nil
	}
	return checkWithGoogle(fName, inMsg.GoogleAuth.IdToken, remoteIP, fw, args, log)
}

// HandleRestfulGoogleAuth ...
func HandleRestfulGoogleAuth(fName string, r *http.Request, preauthUserIDs map[string]time.Time,
	fw *firewall.Checker, args *arguments.Args, log *logging.Logger) (userid string, err error) {
	fName += " (googleAuth)"
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	err = fw.CheckFirewall(firewall.GOOGLEAUTH, fName, remoteIP)
	if log.CheckError(err, fName, "0DDB3B") {
		return userid, fmt.Errorf("Unable to complete request (code 0DDB3B)")
	}
	b, err := ioutil.ReadAll(io.LimitReader(r.Body, 128*int64(units.Kilobyte)))
	if log.CheckError(err, fName, "00A7AF") {
		return userid, fmt.Errorf("Unable to complete request (code 00A7AF)")
	}
	var inMsg clientmessage.Request
	proto.Unmarshal(b, &inMsg)
	if err == http.ErrBodyReadAfterClose {
		return userid, fmt.Errorf("Unable to complete request (code CC30B5)")
	} else if log.CheckError(err, fName, "FBD3CC") {
		return userid, fmt.Errorf("Unable to complete request (code FBD3CC)")
	}
	log.Dump(inMsg)
	if inMsg.GoogleAuth == nil || inMsg.GoogleAuth.UserID == "" || inMsg.GoogleAuth.IdToken == "" {
		return userid, fmt.Errorf("Unable to complete request (code B94319)")
	}
	_, exists := preauthUserIDs[inMsg.GoogleAuth.UserID]
	if exists {
		return inMsg.GoogleAuth.UserID, nil
	}
	return checkWithGoogle(fName, inMsg.GoogleAuth.IdToken, remoteIP, fw, args, log)
}
