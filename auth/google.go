package auth

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/alecthomas/units"
	"github.com/demskie/hyperionfive/clientmessage"
	"github.com/demskie/hyperionfive/firewall"
	"github.com/demskie/logging"
	"github.com/gogo/protobuf/proto"

	"github.com/gorilla/websocket"
	oauth2 "google.golang.org/api/oauth2/v2"
)

type clientAddr string
type userID string

// GoogleAuth will cache authentication results
type GoogleAuth struct {
	mtx   *sync.Mutex
	log   *logging.Logger
	fw    *firewall.Checker
	ga    string
	cache map[clientAddr]map[userID]time.Time
}

// NewGoogleAuth returns a new authenticator
func NewGoogleAuth(logger *logging.Logger, fwall *firewall.Checker, audience string) *GoogleAuth {
	return &GoogleAuth{
		mtx:   &sync.Mutex{},
		log:   logger,
		fw:    fwall,
		ga:    audience,
		cache: make(map[clientAddr]map[userID]time.Time, 0),
	}
}

// HandleWebsocket is used with websocket connections
func (g *GoogleAuth) HandleWebsocket(fName string, conn *websocket.Conn) (userid string, err error) {
	fName += " (googleAuth)"
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	err = g.fw.CheckFirewall(firewall.GOOGLEAUTH, fName, remoteIP)
	if g.log.CheckError(err, fName, "(code 708BD4)") {
		return userid, errors.New("Unable to complete request (code 708BD4)")
	}
	conn.SetReadLimit(128 * int64(units.Kilobyte))
	msgType, b, err := conn.ReadMessage()
	if g.log.CheckError(err, fName, "(code 83FF64)") {
		return userid, errors.New("Unable to complete request (code 83FF64)")
	}
	if msgType != websocket.BinaryMessage {
		return userid, errors.New("Unable to complete request (code 61F991)")
	}
	return g.checkWithGoogle(fName, remoteIP, b)
}

// HandleRestful is used with an http handler
func (g *GoogleAuth) HandleRestful(fName string, r *http.Request) (userid string, err error) {
	fName += " (googleAuth)"
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	err = g.fw.CheckFirewall(firewall.GOOGLEAUTH, fName, remoteIP)
	if g.log.CheckError(err, fName, "0DDB3B") {
		return userid, fmt.Errorf("Unable to complete request (code 0DDB3B)")
	}
	b, err := ioutil.ReadAll(io.LimitReader(r.Body, 128*int64(units.Kilobyte)))
	if g.log.CheckError(err, fName, "00A7AF") {
		return userid, fmt.Errorf("Unable to complete request (code 00A7AF)")
	}
	return g.checkWithGoogle(fName, remoteIP, b)
}

func (g *GoogleAuth) checkWithGoogle(fName string, remoteIP string, b []byte) (userid string, err error) {
	inMsg := &clientmessage.Request{}
	err = proto.Unmarshal(b, inMsg)
	if g.log.CheckError(err, fName, "(code 88473A)") {
		return userid, errors.New("Unable to complete request (code 88473A)")
	}
	if inMsg.GetGoogleAuth().GetUserID() == "" || inMsg.GetGoogleAuth().GetIdToken() == "" {
		return userid, fmt.Errorf("Unable to complete request (code B94319)")
	}
	g.mtx.Lock()
	_, exists := g.cache[clientAddr(remoteIP)]
	if exists {
		_, exists = g.cache[clientAddr(remoteIP)][userID(inMsg.GoogleAuth.UserID)]
		if exists {
			g.mtx.Unlock()
			return inMsg.GoogleAuth.UserID, nil
		}
	}
	g.mtx.Unlock()
	var oauth2Service *oauth2.Service
	var httpClient = &http.Client{}
	oauth2Service, err = oauth2.New(httpClient)
	if g.log.CheckError(err, fName, "(code 6237F5)") {
		return "", errors.New("Unable to complete request (code 6237F5)")
	}
	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(inMsg.GoogleAuth.IdToken)
	tokenInfo, err := tokenInfoCall.Do()
	if g.log.CheckError(err, fName, "(code C7A568)") {
		return "", errors.New("Unable to complete request (code C7A568)")
	}
	if tokenInfo.Audience != g.ga {
		g.log.Error(fName, fmt.Sprintf("googleAudience (%s) != (%s) remoteIP=%s",
			tokenInfo.Audience, g.ga, remoteIP))
		g.fw.TriggerFirewallException(firewall.GOOGLEAUTH, remoteIP)
		return "", errors.New("Unable to complete request (code 3B8111)")
	}
	g.mtx.Lock()
	_, exists = g.cache[clientAddr(remoteIP)]
	if !exists {
		g.cache[clientAddr(remoteIP)] = map[userID]time.Time{}
	}
	g.cache[clientAddr(remoteIP)][userID(tokenInfo.UserId)] = time.Now()
	g.mtx.Unlock()
	return tokenInfo.UserId, nil
}
