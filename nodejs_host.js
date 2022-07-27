let net = require("net");
let sockAddr = "/tmp/melange.sock";

let client = net.createConnection(sockAddr);

client.on("data", data => {
  let requirePath = data.toString();
  delete require.cache[requirePath];
  let value = require(requirePath);
  let json = JSON.stringify(value);
  client.write(json);
});
