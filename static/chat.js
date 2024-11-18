class ChatApp {
  constructor() {
    this.ws = null;
    this.currentUser = null;
    this.currentRoom = null;

    this.elements = {
      loginModal: document.getElementById("loginModal"),
      chatInterface: document.getElementById("chatInterface"),
      username: document.getElementById("username"),
      room: document.getElementById("room"),
      messageInput: document.getElementById("messageInput"),
      messagesContainer: document.getElementById("messagesContainer"),
      roomDisplay: document.getElementById("roomDisplay"),
      userDisplay: document.getElementById("userDisplay"),
    };

    this.bindEvents();
  }

  bindEvents() {
    // Join chat
    document
      .getElementById("joinButton")
      .addEventListener("click", () => this.joinChat());

    document
      .getElementById("leaveButton")
      .addEventListener("click", () => this.leaveChat());

    document
      .getElementById("sendButton")
      .addEventListener("click", () => this.sendMessage());

    this.elements.messageInput.addEventListener("keypress", (e) => {
      if (e.key === "Enter") this.sendMessage();
    });

    this.elements.username.focus();
  }

  async joinChat() {
    const username = this.elements.username.value.trim();
    const room = this.elements.room.value.trim();

    if (!username || !room) {
      this.showError("Please enter both username and room");
      return;
    }

    this.currentUser = username;
    this.currentRoom = room;

    try {
      await this.connectWebSocket(username, room);
      this.updateUIForChat();
      await this.loadChatHistory();
    } catch (error) {
      this.showError("Failed to connect to chat server");
      console.error("Connection error:", error);
    }
  }

  connectWebSocket(username, room) {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(
        `ws://localhost:8080/ws?username=${username}&room=${room}`
      );

      this.ws.onopen = () => resolve();
      this.ws.onerror = (error) => reject(error);

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.appendMessage(message);
        } catch (error) {
          console.error("Message parsing error:", error);
        }
      };

      this.ws.onclose = () => {
        this.showError("Disconnected from chat server");
        this.showLoginModal();
      };
    });
  }

  async loadChatHistory() {
    try {
      const response = await fetch(`/history?room=${this.currentRoom}`);
      if (!response.ok) throw new Error("Failed to fetch chat history");

      const messages = await response.json();
      messages.reverse().forEach((msgStr) => {
        const message = JSON.parse(msgStr);
        this.appendMessage(message);
      });
    } catch (error) {
      console.error("Chat history error:", error);
      this.showError("Failed to load chat history");
    }
  }

  sendMessage() {
    const message = this.elements.messageInput.value.trim();

    if (message && this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(message);
      this.elements.messageInput.value = "";
    }
  }

  appendMessage(message) {
    const messageDiv = document.createElement("div");
    const isSelf = message.username === this.currentUser;

    messageDiv.className = `flex ${isSelf ? "justify-end" : "justify-start"}`;
    messageDiv.innerHTML = `
          <div class="max-w-[70%] ${
            isSelf ? "bg-blue-500 text-white" : "bg-gray-200"
          } rounded-lg px-4 py-2">
              <div class="text-sm ${
                isSelf ? "text-blue-100" : "text-gray-600"
              }">${this.escapeHtml(message.username)}</div>
              <div class="break-words">${this.escapeHtml(message.content)}</div>
              <div class="text-xs ${
                isSelf ? "text-blue-100" : "text-gray-500"
              } mt-1">
                  ${new Date(message.timestamp).toLocaleTimeString()}
              </div>
          </div>
      `;

    this.elements.messagesContainer.appendChild(messageDiv);
    this.scrollToBottom();
  }

  escapeHtml(text) {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }

  scrollToBottom() {
    this.elements.messagesContainer.scrollTop =
      this.elements.messagesContainer.scrollHeight;
  }

  updateUIForChat() {
    this.elements.loginModal.classList.add("hidden");
    this.elements.chatInterface.classList.remove("hidden");
    this.elements.roomDisplay.textContent = `Room: ${this.currentRoom}`;
    this.elements.userDisplay.textContent = `Logged in as: ${this.currentUser}`;
  }

  leaveChat() {
    if (this.ws) {
      this.ws.close();
    }
    this.showLoginModal();
  }

  showLoginModal() {
    this.elements.loginModal.classList.remove("hidden");
    this.elements.chatInterface.classList.add("hidden");
    this.elements.username.value = "";
    this.elements.room.value = "";
    this.currentUser = null;
    this.currentRoom = null;
  }

  showError(message) {
    alert(message); 
  }
}

document.addEventListener("DOMContentLoaded", () => {
  window.chatApp = new ChatApp();
});
