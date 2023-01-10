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
func NewIdentityVerifierCtrl(peer peer.Peer, log *zerolog.Logger) identityverifier {
	return identityverifier{
		peer: peer,
		log:  log,
	}
}

type identityverifier struct {
	peer peer.Peer
	log  *zerolog.Logger
}

func (id identityverifier) VerificationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			connected := id.identityCheckPost(w, r)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "success",
				"statusCode": 200,
				"connected":  connected,
			})
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		default:
			http.Error(w, "forbidden method", http.StatusMethodNotAllowed)
		}
	}
}

//	{
//	    "Name": "XXX",
//	    "Email": "XXX",
//	    "Phone": "XXX",
//	}
func (id identityverifier) identityCheckPost(w http.ResponseWriter, r *http.Request) bool {
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return false
	}

	id.log.Info().Msgf("got the following message: %s", buf)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	res := types.IdentityArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal unicast argument: "+err.Error(),
			http.StatusInternalServerError)
		return false
	}

	connected := id.peer.InitIdentityCheck(res.Name, res.Email, res.Phone)
	return connected
}
