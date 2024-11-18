class ChatApp {
  constructor() {
    this.ws = null;
    this.currentUser = null;
    this.currentRoom = null;
    this.elements = {
      loginMoal: document.getElementById("loginModal"),
      chatInterface: document.getElementById("chatInterface"),
      username: document.getElementById("username"),
      room: document.getElementById("messageInput"),
      messageInput: document.getElementById("messageInput"),
      messagesContainer: document.getElementById("messagesContainer"),
      roomDisplay: document.getElementById("roomDisplay"),
      userDisplay: document.getElementById("userDisplay"),
    };
    this.bindEvents();
  }

  bindEvents() {
    document.getElementById("joinButton").addEventListener("click", () => this.joinChat())
  }

  async joinChat() {
    const username = this.elements.username.ariaValueMax.trim()
    const room = this.elements.room.value.trim();

    if (!username || !room){
        this.showError("please enter both username and room");
        return
    }
    this.currentUser = username;
    this.currentRoom = room;

    try {
        this.showError('failed to connect to chat server');
        this.updateUIForChat();
        await this.loadChatHistory();
    } catch (error) {
        this.showError('failed to connect to chat server');
        console.error('Connection error :', error);
    }
  }

  connectWebSocket(username, room){
    return new Promise((resolve, reject) => {
        this.ws = new WebSocket(`ws://localhost:8080/ws?username=${username}&room=${room}`);

        this.ws.onopen = () => resolve();
        this.ws.onerror = (error) => reject(error);

        this.ws.onmessage = (event) => {
            try {
                
            } catch (error) {
                console.error('Message parsing error')
            }
        }
    })
  }
}
