document.addEventListener("DOMContentLoaded", function () {

    let socket = null;

    let peerConnection = null;
    let dataChannel = null;
    let localStream = null;
    let callActive = false;
    let hasSetRemoteDescription = false;
    let bufferedICECandidates = [];

    let midToParticipant = {};
	let participantToMids = {};
    let participantStreams = {};
    let pendingTracks = {};

	let room = null;

    const joinDiv = document.getElementById("join");
    const chatDiv = document.getElementById("chat");
    const roomTitle = document.getElementById("room-title");
    const messagesDiv = document.getElementById("messages");
    const statusDiv = document.getElementById("status");

    const roomInput = document.getElementById("roomInput");
    const messageInput = document.getElementById("messageInput");
    const joinBtn = document.getElementById("joinBtn");
    const sendBtn = document.getElementById("sendBtn");

    const startCallBtn = document.getElementById("startCallBtn");
    const endCallBtn = document.getElementById("endCallBtn");

    const videoSection = document.getElementById("video-section");
    const videoGrid = document.getElementById("video-grid");

    joinBtn.addEventListener("click", joinRoom);
    sendBtn.addEventListener("click", sendMessage);

    startCallBtn.addEventListener("click", startCall);
    endCallBtn.addEventListener("click", endCall);

    messageInput.addEventListener("keydown", function (e) {
        if (e.key === "Enter") {
            sendMessage();
        }
    });

    function joinRoom() {
        room = roomInput.value.trim();
        if (!room) {
            alert("Room name is required");
            return;
        }

        const protocol = window.location.protocol === "https:" ? "wss" : "ws";
        const wsUrl = `${protocol}://${window.location.host}/ws/chat?roomName=${encodeURIComponent(room)}`;

        socket = new WebSocket(wsUrl);

        socket.onopen = function () {
            joinDiv.style.display = "none";
            chatDiv.style.display = "flex";
            roomTitle.textContent = `Room: ${room}`;
            setStatus("Connected");
        };

        socket.onmessage = onMessage;

        socket.onerror = function () {
            setStatus("WebSocket error");
        };

        socket.onclose = function () {
            setStatus("Disconnected");
            addMessage("⚠️ Connection closed");
        };
    }

    function getOrCreateMediaStreamForParticipantId(participantId) {
		let participantStream = participantStreams[participantId]
		if (participantStream) {
			return participantStream;
		}

		participantStream = new MediaStream();
		participantStreams[participantId] = participantStream;

		addVideoStream(participantStream, participantId, "Participant - " + participantId);
		
		return participantStream;
    }

	async function fetchICEServers() {
		const response = await fetch(`/api/ice-servers?room=${encodeURIComponent(room)}`, {
			method: "GET",
			credentials: "include"
		});

		if (!response.ok) {
			throw new Error("Failed to fetch ICE servers");
		}

		const data = await response.json();
		return data.iceServers;
	}

    async function startCall() {
		if (callActive) return;

		callActive = true;
		
		startCallBtn.style.display = "none";
		endCallBtn.style.display = "inline-block";
		videoSection.style.display = "block";

		localStream = await navigator.mediaDevices.getUserMedia({
			video: true,
			audio: true
		});
		
		addVideoStream(localStream, userID, userFullName, true);
		
		const iceServers = await fetchICEServers();

		console.log("Fetched ICE servers:", iceServers);

		const iceConfig = {
			iceServers: iceServers
		};
		peerConnection = new RTCPeerConnection(iceConfig);
		
		localStream.getTracks().forEach(track => {
				peerConnection.addTrack(track, localStream);
		});

		const max_participants = 10;
		const max_remote_participants = max_participants - 1;
		for (let i=0; i < max_remote_participants; i++) {
			peerConnection.addTransceiver("audio", { direction: "recvonly" });
			peerConnection.addTransceiver("video", { direction: "recvonly" });
		}

		peerConnection.onicecandidate = event => {
			candidate = event.candidate
			if (!candidate) {
				return
			}
			
			socket.send(JSON.stringify({
				Type: "webrtc.ice",
			Data: event.candidate
			}));
		};
		
		peerConnection.ontrack = event => {
			console.log("Track event: ", event);
			const mid = event.transceiver.mid;
			const track = event.track
			console.log("Track event - mid: ", mid, " | Kind: ", track.kind);
			const participantId = midToParticipant[mid];
			if (!participantId) {
				pendingTracks[mid] = track;
				console.log("No participant ID. Track queued.");
				return;
			}
			
			const participantStream = getOrCreateMediaStreamForParticipantId(participantId);

			console.log("Adding track to stream");
			participantStream.addTrack(track);
		};

		dataChannel = peerConnection.createDataChannel("call-info");
		dataChannel.onmessage = event => {
			const msg = JSON.parse(event.data);
			console.log("New data channel message: ", msg);

			const messageType = msg.Type;
			if (messageType === "track-info") {
				const mid = msg.Data.Mid;
				const participantId = msg.Data.ParticipantID;
				midToParticipant[mid] = participantId;
				
				const participantMids = participantToMids[participantId]
				if (participantMids) {
					console.log("Pushed mid for participant")
					participantMids.push(mid);
				} else {
					console.log("Initiated mid array for participant")
					participantToMids[participantId] = [mid]
				}

				const participantStream = getOrCreateMediaStreamForParticipantId(participantId);
				
				const pendingTrack = pendingTracks[mid];
				if (!pendingTrack) {
					console.log("No pending tracks.");
					return;
				}

				console.log("Adding track to stream");
				console.log("Media stream: ", participantStream);
				participantStream.addTrack(pendingTrack);
				delete pendingTracks[mid];
				return;
			}

			if (messageType === "peer-exit-info") {
				const participantId = msg.Data.ParticipantID;
				const mids = participantToMids[participantId];
				if (!mids) {
					return
				}

				mids.forEach(mid => {
					delete midToParticipant[mid];
					delete pendingTracks[mid];
				});

				delete participantToMids[participantId];
				delete participantStreams[participantId];

				removeVideo(participantId);
				
				return;
			}
		}

		peerConnection.onconnectionstatechange = () => {
			connectionState = peerConnection.connectionState
			if (connectionState === "disconnected" || connectionState === "failed" || connectionState === "closed") {
				endCall();
			}
		};

		const offer = await peerConnection.createOffer();
		await peerConnection.setLocalDescription(offer);

		socket.send(JSON.stringify({
			Type: "webrtc.offer",
			Data: { sdp: offer.sdp }
		}));
    }

    function endCall() {
        if (!callActive) return;

		callActive = false;

		startCallBtn.style.display = "inline-block";
		endCallBtn.style.display = "none";
		videoSection.style.display = "none";
		
		if (peerConnection) {
			peerConnection.getSenders().forEach(sender => {
				if (sender.track) sender.track.stop();
			});
			peerConnection.close();
			peerConnection = null;
		}

		if (dataChannel) {
			dataChannel = null;
		}

		midToParticipant = {};
		pendingTracks = {};
		participantStreams = {}

		hasSetRemoteDescription = false;
		bufferedICECandidates = [];
		
		if (localStream) {
			localStream.getTracks().forEach(track => track.stop());
			localStream = null;
		}
		
		videoGrid.innerHTML = "";

		// socket.send(JSON.stringify({
		// 	Type: "webrtc.peer_left"
		// }));
    }

    function addVideoStream(stream, id, label, muted = false) {
	    if (document.getElementById("video-" + id)) return;
	    const wrapper = document.createElement("div");
	    wrapper.className = "video-wrapper";
	    wrapper.id = "video-" + id;

	    const video = document.createElement("video");
	    video.srcObject = stream;
	    video.autoplay = true;
	    video.playsInline = true;
	    video.muted = muted;

	    const nameLabel = document.createElement("div");
	    nameLabel.className = "video-label";
	    nameLabel.textContent = label;
	    
	    wrapper.appendChild(video);
	    wrapper.appendChild(nameLabel);
	    videoGrid.appendChild(wrapper);
    }

    function removeVideo(id) {
	    const el = document.getElementById("video-" + id);

	    if (el) el.remove();
    }

    function sendMessage() {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            return;
        }

        const msg = messageInput.value.trim();
        if (!msg) {
            return;
        }

        socket.send(JSON.stringify({
	    Type: "chat.message",
	    Data: {
		Message: msg
	    }
	}));
        messageInput.value = "";
    }

    function formatTime(isoString) {
	const date = new Date(isoString);
	return date.toLocaleTimeString([], {
		hour: "2-digit",
		minute: "2-digit"
	});
    }

    function addMessage(senderID, senderFullName, message, sentAt) {
	    const wrapper = document.createElement("div");
	    wrapper.className = "message";

	    if (userID === senderID) {
		    wrapper.classList.add("own");
	    }

	    const nameSpan = document.createElement("span");
	    nameSpan.className = "user-full-name";
	    nameSpan.textContent = senderFullName;

	    const separatorSpan = document.createElement("span");
	    separatorSpan.className = "message-separator";
	    separatorSpan.textContent = ": ";

	    const messageSpan = document.createElement("span");
	    messageSpan.className = "message-text";
	    messageSpan.textContent = message;

	    const timeSpan = document.createElement("span");
	    timeSpan.className = "message-time";
	    timeSpan.textContent = formatTime(sentAt);

	    wrapper.appendChild(nameSpan);
	    wrapper.appendChild(separatorSpan);
	    wrapper.appendChild(messageSpan);
	    wrapper.appendChild(timeSpan);

	    messagesDiv.appendChild(wrapper);
	    messagesDiv.scrollTop = messagesDiv.scrollHeight;
    }

    function renderSystemMessage(text) {
	    const div = document.createElement("div");
	    div.className = "system-message";
	    div.textContent = text;
	    messagesDiv.appendChild(div);
	    messagesDiv.scrollTop = messagesDiv.scrollHeight;
    }

    async function onMessage(event) {
	    const payload = JSON.parse(event.data);
	    const payloadType = payload.Type;
	    const data = payload.Data;
	    switch (payloadType) {
		    case "chat.message":
			    addMessage(data.User.ID, data.User.FullName, data.Message, data.Meta.SentAt);
			    break;

		    case "user.join":
			    renderSystemMessage(`${data.User.FullName} joined`);
			    break;

		    case "user.leave":
			    renderSystemMessage(`${data.User.FullName} left`);
				removeVideo(data.userID);
			    break;
		    
		    case "webrtc.answer":
			    if (!peerConnection) {
			        break;
			    }
			    await peerConnection.setRemoteDescription({
				type: "answer",
				sdp: data.sdp
			    });

			    hasSetRemoteDescription = true;
				
			    for (const c of bufferedICECandidates) {
				await peerConnection.addIceCandidate(c);
			    }

			    bufferedICECandidates = [];

			    break;
		    
		    case "webrtc.ice":
		        if (peerConnection) {
			    try {
				if (hasSetRemoteDescription) {
					await peerConnection.addIceCandidate(data);
					break;
				}
					
				bufferedICECandidates.push(data)

			    } catch (err) {
				console.log("data: ", data);
				console.error("ICE error", err);
			    }
			}
			break;
		    
		    // case "webrtc.peer_left":
			// break;
	    }
    }

    function setStatus(text) {
        statusDiv.textContent = text;
    }

});
