/* eslint-env browser */

const pc = new RTCPeerConnection({
  iceServers: [
    {
      urls: "stun:stun.l.google.com:19302",
    },
  ],
});

const log = (msg) => {
  document.getElementById("logs").innerHTML = msg + "<br>";
};

const sendChannel = pc.createDataChannel("foo");
sendChannel.onclose = () => console.log("sendChannel has closed");
sendChannel.onopen = () => console.log("sendChannel has opened");
var count = 0;
const startTime = new Date().getTime();
sendChannel.onmessage = (e) => {
  if (count % 1000 == 0) {
    let endTime = new Date().getTime();
    const timeElapsed = (endTime - startTime) / 1000;
    log("speed = " + count / timeElapsed + " mgs/s " + "count " + count + "\n");
  }
  document.getElementById("messages").innerHTML = e.data + "<br>";
  count += 1;
};

pc.oniceconnectionstatechange = (e) => log(pc.iceConnectionState);
pc.onicecandidate = (event) => {
  if (event.candidate === null) {
    // Send offer message to the server to start WebRTC data channel exchange
    socket.send(btoa(JSON.stringify(pc.localDescription)));
  }
};

var socket = new WebSocket("ws://localhost:8081/websocket");

// Event listener when receiving messages from the server
socket.addEventListener("message", (event) => {
  console.log(`Message from server: ${JSON.parse(atob(event.data))}`);
  try {
    pc.setRemoteDescription(JSON.parse(atob(event.data)));
  } catch (e) {
    alert(e);
  }
  // Handle the received message from the server here
});

// Event listener when the WebSocket is opened
socket.addEventListener("open", () => {
  console.log("WebSocket connection established.");
});

// Event listener when the WebSocket is closed
socket.addEventListener("close", (event) => {
  console.log(`WebSocket disconnected with code ${event.code}`);
});

// Event listener when there's an error with the WebSocket
socket.addEventListener("error", (error) => {
  console.error(`WebSocket error: ${error.message}`);
});

pc.onnegotiationneeded = (e) =>
  pc
    .createOffer()
    .then((d) => pc.setLocalDescription(d))
    .catch(log);

window.sendMessage = () => {
  const message = document.getElementById("message").value;
  if (message === "") {
    return alert("Message must not be empty");
  }

  sendChannel.send(message);
};

window.startSession = () => {
  const sd = document.getElementById("remoteSessionDescription").value;
  if (sd === "") {
    return alert("Session Description must not be empty");
  }

  try {
    pc.setRemoteDescription(JSON.parse(atob(sd)));
  } catch (e) {
    alert(e);
  }
};
