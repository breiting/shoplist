async function checkHealth() {
  const out = document.getElementById("healthOut");
  out.textContent = "…";
  try {
    const res = await fetch("/healthz", { cache: "no-store" });
    const txt = await res.text();
    out.textContent = `${res.status} ${res.statusText}\n${txt}`;
  } catch (e) {
    out.textContent = String(e);
  }
}

function registerServiceWorker() {
  if (!("serviceWorker" in navigator)) return;
  navigator.serviceWorker.register("/sw.js").catch(() => {
    // Keep silent; PWA is optional.
  });
}

document.addEventListener("DOMContentLoaded", () => {
  document.getElementById("btnHealth").addEventListener("click", checkHealth);
  registerServiceWorker();
});
