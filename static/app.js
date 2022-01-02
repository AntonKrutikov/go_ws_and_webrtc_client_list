// WebSocket
let ws_section = document.querySelector('section.ws > .inner')

ws = new WebSocket("ws://localhost:8080/ws")
ws.onopen = () => {
    console.log('WebSocket connected')
}

ws.onclose = () => {
    console.log('WebSocket disconnected')
}

ws.onmessage = (ws_message) => {
    let m = JSON.parse(ws_message.data)
    
    if (m.type == 'client_list') {
        m.body = JSON.parse(m.body)
        ws_section.replaceChildren()

        m.body.forEach(ip => {
            let div = document.createElement('div')
            div.innerText = ip
            ws_section.appendChild(div)
        })
    }

    if (m.type == 'new_client') {
        let div = document.createElement('div')
        div.innerText = m.body
        ws_section.appendChild(div)
    }

    if (m.type == 'client_leave') {
        ws_section.querySelectorAll('div').forEach(div => {
            if (div.innerText == m.body) {
                ws_section.removeChild(div)
            }
        })
    }
}


// WebRTC
let webrtc_section = document.querySelector('section.webrtc > .inner')

let pc = new RTCPeerConnection({
    iceServers: [] //stun, turn part ommited
})
let pc_datachannel = pc.createDataChannel('')

pc_datachannel.onopen = () => {
    console.log('DataChannel oppened')
}

pc_datachannel.onclose = () => {
    console.log('DataChannel closed')
}

pc_datachannel.onmessage = (data_m) => {
    let m = JSON.parse(data_m.data)

    if (m.type == 'client_list') {
        m.body = JSON.parse(m.body)
        webrtc_section.replaceChildren()

        m.body.forEach(ip => {
            let div = document.createElement('div')
            div.innerText = ip
            webrtc_section.appendChild(div)
        })
    }

    if (m.type == 'new_client') {
        let div = document.createElement('div')
        div.innerText = m.body
        webrtc_section.appendChild(div)
    }

    if (m.type == 'client_leave') {
        webrtc_section.querySelectorAll('div').forEach(div => {
            if (div.innerText == m.body) {
                webrtc_section.removeChild(div)
            }
        })
    }
}

pc.onconnectionstatechange = (e) => {
}

pc.onnegotiationneeded = async (e) => {
    let local_offer = await pc.createOffer()
    pc.setLocalDescription(local_offer)

    let response = await fetch('/webrtc/sdp', {
        method: 'POST',
        body: local_offer.sdp
    })
    let answer = await response.json()
    await pc.setRemoteDescription(new RTCSessionDescription(answer))
}