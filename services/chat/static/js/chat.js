document.addEventListener("DOMContentLoaded", function () {

    let socket = null;

    const joinDiv = document.getElementById("join");
    const chatDiv = document.getElementById("chat");
    const roomTitle = document.getElementById("room-title");
    const messagesDiv = document.getElementById("messages");
    const statusDiv = document.getElementById("status");

    const roomInput = document.getElementById("roomInput");
    const messageInput = document.getElementById("messageInput");
    const joinBtn = document.getElementById("joinBtn");
    const sendBtn = document.getElementById("sendBtn");

    joinBtn.addEventListener("click", joinRoom);
    sendBtn.addEventListener("click", sendMessage);

    messageInput.addEventListener("keydown", function (e) {
        if (e.key === "Enter") {
            sendMessage();
        }
    });

    function joinRoom() {
        const room = roomInput.value.trim();
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

    function sendMessage() {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            return;
        }

        const msg = messageInput.value.trim();
        if (!msg) {
            return;
        }

        socket.send(msg);
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

    function onMessage(event) {
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
			    break;
	    }
    }

    function setStatus(text) {
        statusDiv.textContent = text;
    }

});
