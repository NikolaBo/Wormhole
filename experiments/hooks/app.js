const express = require("express");
const os = require("os")
const app = express();

const DEFAULT_PORT = 8000;

const kvStore = new Map();

console.log("Server starting");
app.use(express.json());

app.get("/test", (req, res) => {
  console.log("Received request from " + req.headers.host);
  res.send("Test endpoint on " + os.hostname() + "\n");
});

app.post("/put", (req, res) => {
  kvStore.set(req.body.key, req.body.value);
  res.send(req.body.key + " set to " + req.body.value + "\n");
});

app.get("/get", (req, res) => {
  res.send(kvStore.get(req.query.key) + "\n");
});

function shutdown() {
  console.log("Server shutting down");
  process.exit(0);
}

process.on("SIGTERM", shutdown);

const PORT = process.env.PORT || DEFAULT_PORT;
app.listen(PORT);