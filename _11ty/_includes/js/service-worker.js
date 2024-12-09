let isRefreshing = false;

const refresh = () => {
  if (isRefreshing) {
    return null;
  }

  isRefreshing = true;
  window.location.reload();
};

window.addEventListener("load", async () => {
  if ("serviceWorker" in navigator) {
    const IS_INITIAL = !navigator.serviceWorker.controller;
    try {
      const reg = await navigator.serviceWorker.register("/sw.js");

      !IS_INITIAL &&
        reg.addEventListener("updatefound", () => {
          renderLoader();
          const worker = reg.installing;

          worker.addEventListener("statechange", () => {
            worker.state === "installed" &&
              worker.postMessage({ action: "skipWaiting" });
          });
        });

      !IS_INITIAL &&
        navigator.serviceWorker.addEventListener("controllerchange", refresh);
    } catch (e) {
      console.error("Service worker registration failed", e);
    }
  }
});
