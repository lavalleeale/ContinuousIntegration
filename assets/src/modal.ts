const modal = document.getElementById("modal")!;

globalThis.showModal = () => {
  modal.style.display = "block";
};

modal.onclick = function (e) {
  if (e.target === modal) {
    modal.style.display = "none";
  }
};
