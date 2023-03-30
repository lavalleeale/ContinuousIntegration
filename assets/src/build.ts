(() => {
  if (!document.querySelector(".bg-yellow-500")) {
    return;
  }

  const ws = new WebSocket(
    `ws://localhost:8080/build/${window.location.href.substring(
      window.location.href.lastIndexOf("/") + 1
    )}/containerStatus`
  );

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    const container = document.getElementById(data.id)!;
    container.classList.remove("bg-yellow-500", "bg-green-500", "bg-red-500");
    if (data.code === 0) {
      container.classList.add("bg-green-500");
      container.innerHTML = "✓";
    } else {
      container.classList.add("bg-red-500");
      container.innerHTML = "X";
    }
  };
})();
