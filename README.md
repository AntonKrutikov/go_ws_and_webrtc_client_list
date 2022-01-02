## Example of handling WebSocket and WebRTC connections

Using:

- WebSockets: https://golang.org/x/net/websocket
- WebRTC: https://github.com/pion/webrtc

For both message types are [`client_list`, `new_client`, `client_leave`] sending over ws or webrtc datachannel encoded as JSON.

Client ip stored in `WsClientStore` or `WebRTCClientStore`