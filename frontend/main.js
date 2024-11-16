const localVideo = document.getElementById("localVideo");
const remoteVideo = document.getElementById("remoteVideo");

const startCall = async () => {
  const peerConnection = new RTCPeerConnection();
  const stream = await navigator.mediaDevices.getUserMedia({
    video: true,
    audio: true,
  });
  localVideo.srcObject = stream;

  stream.getTracks().forEach((track) => peerConnection.addTrack(track, stream));
  peerConnection.ontrack = (event) => {
    remoteVideo.srcObject = event.streams[0];
  };

  const offer = await peerConnection.createOffer();
  await peerConnection.setLocalDescription(offer);

  //sending offer to the backend service
  const response = await fetch("http://localhost:8080/offer", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ offer: offer.sdp }),
  });
  const { answer } = await response.json();

  const remoteDesc = new RTCSessionDescription({ type: "answer", sdp: answer });
  await peerConnection.setRemoteDescription(remoteDesc);
};

window.onload = startCall;
