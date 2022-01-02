package main

import (
	"sync"

	"golang.org/x/net/websocket"
)

type WsClients struct {
	Map map[string]*websocket.Conn
	Mu  sync.Mutex
}

func (clients *WsClients) Add(ip string, ws *websocket.Conn) {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	if _, ok := clients.Map[ip]; ok {
		return
	} else {
		clients.Map[ip] = ws
	}
}

func (clients *WsClients) Remove(ip string) {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	if _, ok := clients.Map[ip]; ok {
		delete(clients.Map, ip)
	}
}

func (clients *WsClients) List() []string {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	list := []string{}
	for k := range clients.Map {
		list = append(list, k)
	}
	return list
}

func (clients *WsClients) Broadcast(from *websocket.Conn, msg string) {
	clients.Mu.Lock()
	defer clients.Mu.Unlock()
	for _, ws := range clients.Map {
		if ws != from {
			websocket.Message.Send(ws, msg)
		}
	}
}
