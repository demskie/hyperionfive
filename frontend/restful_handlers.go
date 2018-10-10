package frontend

import (
	"net"
	"net/http"
	"sync"

	"github.com/alecthomas/units"
	"github.com/demskie/hyperionfive/auth"
	"github.com/demskie/hyperionfive/frontendmessage"
	"github.com/gogo/protobuf/proto"
)

var serverInfoBufferPool = sync.Pool{
	New: func() interface{} {
		return proto.NewBuffer([]byte{})
	},
}

func (s *Service) handleClientToFrontend(w http.ResponseWriter, r *http.Request) {
	const fName = "(clientToFrontend)"
	s.RLock()
	defer s.RUnlock()
	r.Body = http.MaxBytesReader(w, r.Body, 1*int64(units.Megabyte))
	httpRemoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	possibleProfiles := s.addrToProfiles[httpRemoteIP]
	pb := serverInfoBufferPool.Get().(proto.Buffer)
	pb.Reset()
	defer func() {
		serverInfoBufferPool.Put(pb)
	}()
	_, err := auth.HandleRestfulGoogleAuth(fName, r, possibleProfiles, fwall, args, log)
	if err != nil {
		err = pb.Marshal(
			&frontendmessage.ServerInfo{
				ErrorMessage: err.Error(),
			},
		)
	} else {
		err = pb.Marshal(
			&frontendmessage.ServerInfo{
				ErrorMessage: "",
				InstanceName: s.instanceName,
				Version:      s.version,
				AllServers:   s.allServers,
				YourServers:  s.allServers, // TODO: actually implement
			},
		)
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(pb.Bytes())
	} else {
		http.Error(w, "encountered protobuf encoding error", http.StatusProcessing)
	}
}
