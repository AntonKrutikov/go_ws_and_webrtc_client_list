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

type Peer struct {
	Connection    *webrtc.PeerConnection
	IceCandidates struct {
		NewLocal []webrtc.ICECandidateInit
	}
}

type Session struct {
	ID   string
	Peer *Peer
}

var SessionStore = struct {
	Map map[string]*Session
	Mu  sync.Mutex
}{
	Map: map[string]*Session{},
	Mu:  sync.Mutex{},
}

func login(w http.ResponseWriter, r *http.Request) {
	sessionId := pseudo_uuid()
	SessionStore.Mu.Lock()
	SessionStore.Map[sessionId] = &Session{
		ID: sessionId,
		Peer: &Peer{
			Connection: nil,
			IceCandidates: struct {
				NewLocal []webrtc.ICECandidateInit
			}{},
		},
	}
	SessionStore.Mu.Unlock()

	w.Write([]byte(sessionId))
}

func webrtcHandler(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Query().Get("session")
	if sessionId == "" {
		w.WriteHeader(401)
		return
	}

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
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Println(err)
	}

	//Add to session
	SessionStore.Mu.Lock()
	SessionStore.Map[sessionId].Peer.Connection = peerConnection
	SessionStore.Mu.Unlock()

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		SessionStore.Mu.Lock()
		if candidate != nil {
			SessionStore.Map[sessionId].Peer.IceCandidates.NewLocal = append(SessionStore.Map[sessionId].Peer.IceCandidates.NewLocal, candidate.ToJSON())
		}
		SessionStore.Mu.Unlock()
	})

	// Set datachannel handler
	// Found a strange behaviour as *webrtc.DataChannel return always nil on .Transport() - pass pc too
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) { onDataChannel(peerConnection, d) })

	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		fmt.Println("REMOTE SDP ERROR", err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		fmt.Println(err)
	}

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		fmt.Println(err)
	}

	response, _ := json.Marshal(peerConnection.LocalDescription())
	w.Write(response)
}

func iceHandler(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Query().Get("session")
	if sessionId == "" {
		w.WriteHeader(401)
		return
	}
	if r.Method == "GET" {
		// Return new collected candidates from last request
		SessionStore.Mu.Lock()
		candidates := SessionStore.Map[sessionId].Peer.IceCandidates.NewLocal
		fmt.Printf("\n\nLocal ice candidates gathered from last request:\n")
		for _, c := range candidates {
			fmt.Println(c.Candidate)
		}
		fmt.Printf("--END--\n")
		SessionStore.Map[sessionId].Peer.IceCandidates.NewLocal = nil
		SessionStore.Mu.Unlock()

		body, _ := json.Marshal(candidates)
		w.Write(body)
	}
	if r.Method == "POST" {
		candidates := []webrtc.ICECandidateInit{}
		json.NewDecoder(r.Body).Decode(&candidates)

		fmt.Printf("\n\nRemote ice candidates portion recieved:\n")
		for _, c := range candidates {
			fmt.Println(c.Candidate)
		}
		fmt.Printf("--END--\n\n")

		SessionStore.Mu.Lock()
		con := SessionStore.Map[sessionId].Peer.Connection
		SessionStore.Mu.Unlock()

		for _, c := range candidates {
			con.AddICECandidate(c)
		}
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/login", login)
	http.Handle("/ws", websocket.Handler(wsHandler))
	http.HandleFunc("/webrtc/sdp", webrtcHandler)
	http.HandleFunc("/webrtc/ice", iceHandler)
	http.ListenAndServe("0.0.0.0:8080", nil)
}
