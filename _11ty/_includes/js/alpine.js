function urlB64ToUint8Array(base64String) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

document.addEventListener("alpine:init", () => {
  Alpine.data("subscription", () => ({
    isDisabled: true,
    isSubscribed: false,
    reg: null,
    async init() {
      if (!("serviceWorker" in navigator)) {
        console.warn("Service workers are not supported in this browser.");
        return;
      }

      if (!("showNotification" in ServiceWorkerRegistration.prototype)) {
        console.warn("Notifications are not supported in this browser.");
        return;
      }

      if (!("PushManager" in window)) {
        console.warn("Push notifications are not supported in this browser.");
        return;
      }

      const reg = await navigator.serviceWorker.ready;

      this.reg = reg;
      this.isDisabled = false;
    },
    async subscribe() {
      if (!this.reg) {
        return;
      }

      const options = {
        userVisibleOnly: true,
        applicationServerKey: urlB64ToUint8Array(window.VAPID_PUBLIC_KEY),
      };

      const subscription = await this.reg.pushManager.subscribe(options);

      const res = await fetch("/demo/subscribe", {
        method: "POST",
        body: JSON.stringify(subscription),
        headers: { "content-type": "application/json" },
      });

      if (res.status === 201) {
        this.isSubscribed = true;
        return;
      }

      console.error("Failed to register push subscription!");

      // TODO: show notification
    },
    async unsubscribe() {
      if (!this.reg) {
        return;
      }

      const subscription = await this.reg.pushManager.getSubscription();

      if (!subscription) {
        this.isSubscribed = false;
        return;
      }

      await subscription.unsubscribe();
      this.isSubscribed = false;

      // TODO: show notification
    },
  }));
});
