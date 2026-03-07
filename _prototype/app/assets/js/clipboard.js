export default function (rootEl) {
  rootEl.querySelectorAll("[data-clipboard]").forEach((el) => {
    el.addEventListener("click", () => {
      navigator.clipboard.writeText(el.dataset.clipboard).then(() => {
        let icon = el.querySelector(".if");
        let text = el.querySelector(".btn-text");
        let origBtnClass = el.className;
        let origIconClass = icon.className;
        let origTextClass = text.className;
        let origText = text.innerText;

        el.classList.remove("btn-outline-secondary");
        el.classList.add("btn-outline-success");

        icon.classList.remove("if-copy", "text-muted");
        icon.classList.add("if-check", "text-success");

        text.classList.remove("text-muted");
        text.classList.add("text-success");
        text.setAttribute("aria-live", "polite");
        text.innerText = "Copied";

        setTimeout(function () {
          el.className = origBtnClass;
          icon.className = origIconClass;
          text.className = origTextClass;
          text.innerText = origText;
        }, 1500);
      });
    });
  });
}
