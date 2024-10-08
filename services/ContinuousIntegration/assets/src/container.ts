import Convert from "ansi-to-html";
import ReconnectingWebSocket from "reconnecting-websocket";

window.addEventListener("pageshow", () => {
  const convert = new Convert({
    newline: true,
    stream: false,
    fg: "#FFF",
    bg: "#0F172A",
  });
  const code = document.getElementById("code")!;
  const log = document.getElementById("log")!;
  log.innerHTML = convert.toHtml(log.innerHTML);
  if (
    !code.innerText.includes("Running") &&
    !code.innerText.includes("Waiting")
  ) {
    return;
  }

  const ws = new ReconnectingWebSocket(
    `${window.location.protocol.replace("http", "ws")}//${
      window.location.host
    }/build/${window.location.href.split("/")[4]}/container/${
      window.location.href.split("/")[6]
    }/log`
  );

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === "log") {
      const wantsBottm =
        window.innerHeight + Math.round(window.scrollY) >=
        document.body.offsetHeight;
      log.innerHTML += convert.toHtml(data.log);
      if (wantsBottm) {
        window.scrollTo(0, document.body.scrollHeight);
      }
    } else {
      code.innerText = code.innerText = `Exit Code: ${data.code}`;
      ws.close();
    }
  };
});
