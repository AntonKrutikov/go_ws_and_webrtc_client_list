package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
	"golang.org/x/net/websocket"
)

type Message struct {
	Type string `json:"type"`
	Body string `json:"body"`
}

var WsClientStore = &WsClients{
	Map: map[string]*websocket.Conn{},
	Mu:  sync.Mutex{},
}

var WebRTCClientStore = &WebRTCClients{
	Map: map[string]*webrtc.DataChannel{},
	Mu:  sync.Mutex{},
}

func wsHandler(ws *websocket.Conn) {
	ctx := ws.Request().Context()
	// Store IP on connection and remove on close
	WsClientStore.Add(ws.Request().RemoteAddr, ws)
	defer func() {
		WsClientStore.Remove(ws.Request().RemoteAddr)
		msg := Message{
			Type: "client_leave",
			Body: ws.Request().RemoteAddr,
		}
		response, _ := json.Marshal(&msg)
		WsClientStore.Broadcast(ws, string(response))
	}()

	// Send array of clients ip to client
	list, _ := json.Marshal(WsClientStore.List())
	msg := Message{
		Type: "client_list",
		Body: string(list),
	}
	response, _ := json.Marshal(&msg)
	err := websocket.Message.Send(ws, string(response))
	if err != nil {
		fmt.Println(err)
	}

	// Broadcast to others
	msg = Message{
		Type: "new_client",
		Body: ws.Request().RemoteAddr,
	}
	response, _ = json.Marshal(&msg)
	WsClientStore.Broadcast(ws, string(response))

	// Not close connection, but wait for incoming fragments
	select {
	case <-ctx.Done():
		break
	default:
		var message string
		websocket.Message.Receive(ws, &message)
	}
}

func webrtcHandler(w http.ResponseWriter, r *http.Request) {
	sdp, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
	}
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(sdp),
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302?transport=tcp"},
			},
		},
	} //ommited for simplicity
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Println(err)
	}

	// Set datachannel handler
	// Found a strange behaviour as *webrtc.DataChannel return always nil on .Transport() - pass pc too
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) { onDataChannel(peerConnection, d) })

	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		fmt.Println(i)
	})

	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		fmt.Println("REMOTE SDP ERROR", err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		fmt.Println(err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		fmt.Println(err)
	}

	<-gatherComplete

	response, _ := json.Marshal(peerConnection.LocalDescription())
	w.Write(response)
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.Handle("/ws", websocket.Handler(wsHandler))
	http.HandleFunc("/webrtc/sdp", webrtcHandler)
	http.ListenAndServe("0.0.0.0:8080", nil)
}
