// WebSocket
let ws_section = document.querySelector('section.ws > .inner')

let ws = new WebSocket(`ws://${window.location.host}/ws`)
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

//pseudo login
let response = await fetch('/login')
let session = await response.text()

let pc = new RTCPeerConnection({
    iceServers: [
        {
            urls: ["stun:stun.l.google.com:19302"]
        },

    ], //stun, turn part ommited
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


let localIce = []

// When new browser ice candidate gathered - add it to localIce array, we will send this array after initial sdp
pc.onicecandidate = async (e) => {
    if (e.candidate !== null) {
        localIce.push({
            candidate: e.candidate.candidate,
            sdpMid: e.sdpMid,
            sdpMLineIndex: e.sdpMLineIndex,
            usernameFragment: e.usernameFragment
        })
    }
}

// Get candidates from server until connected state
async function getIce() {
    if (pc.connectionState != 'connected') {
        let response = await fetch(`/webrtc/ice?session=${session}`)
        let candidates = await response.json()

        candidates.forEach(c => {
            let ice = new RTCIceCandidate(c)
            pc.addIceCandidate(ice)
        })
        setTimeout(getIce, 1000)
    }
}

// Send new candidates untill full gathered. Store sended to not resend.
let iceSended = []
async function sendIce() {
    let part = localIce.filter(i => iceSended.indexOf(i) == -1)
    iceSended.push(...part)
    if (pc.iceGatheringState != 'complete') {
        await fetch(`/webrtc/ice?session=${session}`, {
            method: 'POST',
            body: JSON.stringify(localIce)
        })
        setTimeout(sendIce, 1000)
    }
}

pc.onnegotiationneeded = async (e) => {
    let local_offer = await pc.createOffer()
    pc.setLocalDescription(local_offer)

    let response = await fetch(`/webrtc/sdp?session=${session}`, {
        method: 'POST',
        body: local_offer.sdp
    })
    let answer = await response.json()
    await pc.setRemoteDescription(new RTCSessionDescription(answer))
    // We wait offer-answer exchange because our fake session mechanism (wait server peerConnection 100% crreated and append to session)
    // Because ice gathering starts immideatly after peerConnection instance created
    // In real app session exists and websockets transport with message order can be used
    getIce()
    sendIce()
}