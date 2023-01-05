package controller

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/gui/httpnode/types"
	"go.dedis.ch/cs438/peer"
)

// NewIdentityCtrl returns a new initialized service controller.
func NewIdentityCtrl(peer peer.Peer, log *zerolog.Logger) identityctrl {
	return identityctrl{
		peer: peer,
		log:  log,
	}
}

type identityctrl struct {
	peer peer.Peer
	log  *zerolog.Logger
}

func (i identityctrl) IdentityCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			i.identityCheckPost(w, r)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			return
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
			return
		}
	}
}

//	{
//	    "Name": "XXX",
//	    "Email": "XXX",
//	    "Phone": "XXX",
//	}
func (i identityctrl) identityCheckPost(w http.ResponseWriter, r *http.Request) {
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	i.log.Info().Msgf("got the following message: %s", buf)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	res := types.IdentityArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal unicast argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	i.peer.InitIdentityCheck(res.Name, res.Email, res.Phone)
}
