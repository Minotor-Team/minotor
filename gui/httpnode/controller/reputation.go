package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/gui/httpnode/types"
	"go.dedis.ch/cs438/peer"
)

// NewIdentityCtrl returns a new initialized service controller.
func NewReputationCtrl(node peer.Peer, log *zerolog.Logger) reputationctrl {
	return reputationctrl{
		node: node,
		log:  log,
	}
}

type reputationctrl struct {
	node peer.Peer
	log  *zerolog.Logger
}

// handle the likes post
func (m messaging) LikeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.likePost(w, r)
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

// handle the dislikes post
func (m messaging) DisLikeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			m.dislikePost(w, r)
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

func (m messaging) likePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("like got the following message: %s", buf)

	res := types.ReputationArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal like argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	actualScore := res.NbLikes - res.NbDisLikes
	fmt.Println(res.UserID)
	fmt.Println(res.NbLikes)
	fmt.Println(res.NbDisLikes)
	err = m.node.InitReputationCheck("", actualScore+1)
	if err != nil {
		http.Error(w, "failed to init reputation consensus: "+err.Error(),
			http.StatusInternalServerError)
		return
	}
}
func (m messaging) dislikePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	m.log.Info().Msgf("dislike got the following message: %s", buf)

	res := types.ReputationArgument{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		http.Error(w, "failed to unmarshal dislike argument: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	actualScore := res.NbLikes - res.NbDisLikes

	m.node.InitReputationCheck("", actualScore-1)

}
