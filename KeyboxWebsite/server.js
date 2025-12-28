const { SerialPort } = require("serialport");
const { ReadlineParser } = require("@serialport/parser-readline");
const { Server } = require("socket.io");
const http = require("http");
const express = require("express");

const app = express();
const server = http.createServer(app);
const io = new Server(server);

app.use(express.json());
app.get("/", (req, res) => {
  res.sendFile(__dirname + "/view/index.html");
});

io.on("connection", (socket) => {
  console.log("Socket.IO client connected");
  socket.on("disconnect", () => {
    console.log("Socket.IO client disconnected");
  });
});

server.listen(3000, () => {
  console.log("Server listening on port 3000");
});

const port = new SerialPort({
  path: "COM5",
  baudRate: 9600,
});

const parser = port.pipe(new ReadlineParser({ delimiter: "\r\n" }));

// tangkap data dari arduino
parser.on("data", (data) => {
  console.log(data);
  io.emit("data", { data: data });
});

app.post("/open", (req, res) => {
  const data = req.body.data;
  port.write(data, (err) => {
    if (err) {
      console.log("err: ", err);
      res.status(500).json({ message: "error" });
    }
    console.log("data terkirim ->", data);
    res.end();
  });
});
