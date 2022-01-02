package main

import (
	"sync"

	"golang.org/x/net/websocket"
)

type WsClients struct {
	Map map[string]*websocket.Conn
	Mu  sync.Mutex
}

func (wsc *WsClients) Add(ip string, ws *websocket.Conn) {
	wsc.Mu.Lock()
	defer wsc.Mu.Unlock()
	if _, ok := wsc.Map[ip]; ok {
		return
	} else {
		wsc.Map[ip] = ws
	}
}

func (wsc *WsClients) Remove(ip string) {
	wsc.Mu.Lock()
	defer wsc.Mu.Unlock()
	if _, ok := wsc.Map[ip]; ok {
		delete(wsc.Map, ip)
	}
}

func (wsc *WsClients) List() []string {
	wsc.Mu.Lock()
	defer wsc.Mu.Unlock()
	list := []string{}
	for k := range wsc.Map {
		list = append(list, k)
	}
	return list
}

func (wsc *WsClients) Broadcast(from *websocket.Conn, msg string) {
	wsc.Mu.Lock()
	defer wsc.Mu.Unlock()
	for _, ws := range wsc.Map {
		if ws != from {
			websocket.Message.Send(ws, msg)
		}
	}
}
