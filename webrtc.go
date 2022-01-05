package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pion/webrtc/v3"
)

type WebRTCClients struct {
	Map map[string]*webrtc.DataChannel
	Mu  sync.Mutex
}

func (clients *WebRTCClients) Add(ip string, d *webrtc.DataChannel) {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	if _, ok := clients.Map[ip]; ok {
		return
	} else {
		clients.Map[ip] = d
	}
}

func (clients *WebRTCClients) Remove(ip string) {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	if _, ok := clients.Map[ip]; ok {
		delete(clients.Map, ip)
	}
}

func (clients *WebRTCClients) List() []string {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	list := []string{}
	for k := range clients.Map {
		list = append(list, k)
	}
	return list
}

func (clients *WebRTCClients) Broadcast(from *webrtc.DataChannel, msg string) {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	for _, d := range clients.Map {
		if d != from {
			d.SendText(msg)
		}
	}
}

func onConnectionStateChanged(d *webrtc.DataChannel, pcs webrtc.PeerConnectionState) {
	if pcs == webrtc.PeerConnectionStateClosed {
		d.Close()
	}
}

func onDataChannel(peer *webrtc.PeerConnection, d *webrtc.DataChannel) {
	pairs, err := peer.SCTP().Transport().ICETransport().GetSelectedCandidatePair()
	// Basicly this err can't happen, because if err - connection not exists
	if err != nil {
		fmt.Println(err)
		return
	}

	print("Selected pair: ", pairs.String())

	// Can't return only ip and port, because connection can be related
	WebRTCClientStore.Add(pairs.Remote.String(), d)
	msg := &Message{
		Type: "new_client",
		Body: pairs.Remote.String(),
	}
	response, _ := json.Marshal(msg)
	WebRTCClientStore.Broadcast(d, string(response))

	// DataChannel.Close is not fired if client don't send close signal (close tab fro example) - use state event to detect client leave
	peer.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) { onConnectionStateChanged(d, pcs) })

	d.OnClose(func() {
		WebRTCClientStore.Remove(pairs.Remote.String())
		msg := &Message{
			Type: "client_leave",
			Body: pairs.Remote.String(),
		}
		response, _ := json.Marshal(msg)
		WebRTCClientStore.Broadcast(d, string(response))
	})

	// Register channel opening handling
	d.OnOpen(func() {
		list, _ := json.Marshal(WebRTCClientStore.List())
		msg := &Message{
			Type: "client_list",
			Body: string(list),
		}
		response, _ := json.Marshal(msg)
		err := d.SendText(string(response))
		if err != nil {
			fmt.Println(err)
		}
	})
}
