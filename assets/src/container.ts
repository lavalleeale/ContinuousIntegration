const code = document.getElementById("code")!;
const log = document.getElementById("log")!;
(() => {
  if (!code.innerText.includes("Running")) {
    return;
  }

  const ws = new WebSocket(
    `ws://localhost:8080/build/${
      window.location.href.split("/")[4]
    }/container/${window.location.href.split("/")[6]}/log?token=${
      document.cookie.split("=")[1]
    }`
  );

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === "log") {
      log.innerHTML += data.log;
      window.scrollTo(0, document.body.scrollHeight);
    } else {
      code.innerText = code.innerText.replace("Running", data.code);
      ws.close();
    }
  };
})();
