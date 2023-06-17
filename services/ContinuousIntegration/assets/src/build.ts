
import ReconnectingWebSocket from "reconnecting-websocket";

window.onload = () => {
  var left =
    document.querySelectorAll("[id]").length -
    document.querySelectorAll(".bg-green-500").length;
  const ws = new ReconnectingWebSocket(
    `${window.location.protocol.replace("http","ws")}//${window.location.host}/build/${window.location.href.substring(
      window.location.href.lastIndexOf("/") + 1
    )}/containerStatus`
  );

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    const container = document.getElementById(data.id);
    if (container) {
      container.classList.remove(
        "bg-gray-500",
        "bg-yellow-500",
        "bg-green-500",
        "bg-red-500"
      );
      if (data.type == "create") {
        container.classList.add("bg-yellow-500");
      } else if (data.code === "0") {
        container.classList.add("bg-green-500");
        container.innerHTML = "✓";
        left--;
        if (left === 0) {
          ws.close();
        }
      } else {
        container.classList.add("bg-red-500");
        container.innerHTML = "X";
      }
    }
  };
};
