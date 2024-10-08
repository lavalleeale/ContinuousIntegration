import mermaid from "mermaid";
import ReconnectingWebSocket from "reconnecting-websocket";

window.addEventListener("pageshow", async () => {
  var element = document.querySelector("pre.flowchart");
  const foregroundColor =
    window.matchMedia &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "white"
      : "black";
  mermaid.initialize({
    securityLevel: "loose",
    theme: "base",
    themeVariables: {
      primaryColor: "transparent",
      lineColor: foregroundColor,
      tertiaryTextColor: foregroundColor,
    },
  });
  element.attributes.removeNamedItem("style");
  await mermaid.run({ querySelector: "pre.flowchart" });
  var left = document.querySelectorAll(".bg-gray-500,.bg-yellow-500").length;
  if (left > 0) {
    const ws = new ReconnectingWebSocket(
      `${window.location.protocol.replace("http", "ws")}//${
        window.location.host
      }/build/${window.location.href.substring(
        window.location.href.lastIndexOf("/") + 1
      )}/containerStatus`
    );

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      console.log(data);
      const container = document.getElementById(data.id);
      if (container) {
        container.classList.remove(
          "bg-gray-500",
          "bg-yellow-500",
          "bg-green-500",
          "bg-red-500"
        );
        switch (data.type) {
          case "create":
            container.classList.add("bg-yellow-500");
            break;
          case "die":
            left = document.querySelectorAll(
              ".bg-gray-500,.bg-yellow-500"
            ).length;
            if (data.code === "0") {
              container.classList.add("bg-green-500");
              container.innerHTML = "✓";
              if (left === 0) {
                ws.close();
                document.getElementById("status").innerText = `Status : ${
                  document.querySelectorAll(".bg-red-500").length > 0
                    ? "failure"
                    : "success"
                }`;
              }
            } else {
              container.classList.add("bg-red-500");
              container.innerHTML = "X";
            }
            break;
          default:
            break;
        }
      }
      console.log(left);
    };
  }
});
