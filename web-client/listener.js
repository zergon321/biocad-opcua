var socket = new WebSocket("ws://127.0.0.1:8080/api/measures");
var output = document.getElementById("data-field");
var writeMessage = function(message) {
    output.innerHTML += "<p>service: " + message + "</p>";
}

socket.onopen = function () {
    writeMessage("Connection established");
};

socket.onclose = function (event) {
    if (event.wasClean) {
        writeMessage('Connection closed gracefully');
    } else {
        writeMessage('Connection aborted');
    }

    writeMessage('code: ' + event.code + ' reason: ' + event.reason);
};

socket.onmessage = function(event) {
    output.innerHTML += "<p>data: " + event.data + "</p>";
}

socket.onerror = function(error) {
    writeMessage("error occurred: " + error.message);
}