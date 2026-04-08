function urlB64ToUint8Array(base64String) {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replaceAll("-", "+")
    .replaceAll("_", "/");

  const rawData = globalThis.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.codePointAt(i);
  }
  return outputArray;
}

document.addEventListener("alpine:init", () => {
  Alpine.data("subscription", () => ({
    isDisabled: true,
    isLoading: false,
    isSubscribed: false,
    reg: null,
    toast: null,

    getDeviceId() {
      let deviceId = localStorage.getItem("deviceId");

      if (!deviceId) {
        deviceId = crypto.randomUUID();
        localStorage.setItem("deviceId", deviceId);
      }

      return deviceId;
    },
    showToast(message, duration = 5000) {
      this.toast = message;
      setTimeout(() => {
        this.toast = null;
      }, duration);
    },
    async init() {
      if (!("serviceWorker" in navigator)) {
        console.warn("Service workers are not supported in this browser.");
        return;
      }

      if (!("showNotification" in ServiceWorkerRegistration.prototype)) {
        console.warn("Notifications are not supported in this browser.");
        return;
      }

      if (!("PushManager" in globalThis)) {
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

      this.isLoading = true;

      const options = {
        userVisibleOnly: true,
        applicationServerKey: urlB64ToUint8Array(globalThis.VAPID_PUBLIC_KEY),
      };

      try {
        const subscription = await this.reg.pushManager.subscribe(options);

        const res = await fetch("/demo/subscribe", {
          method: "POST",
          body: JSON.stringify(subscription),
          credentials: "same-origin",
          headers: {
            "content-type": "application/json",
            "x-device-id": this.getDeviceId(),
          },
        });

        if (res.status === 201) {
          this.isSubscribed = true;
          this.isLoading = false;
          return;
        }

        const { errors } = await res.json();
        this.showToast(errors[0]?.title ?? "Subscription failed");
        console.error("Failed to register push subscription!");
      } catch (e) {
        console.error(e);

        this.showToast(e.message || "Something went wrong");
        console.error("Failed to register push subscription!");
      } finally {
        this.isLoading = false;
      }
    },
    async unsubscribe() {
      if (!this.reg) {
        return;
      }

      this.isLoading = true;

      try {
        const subscription = await this.reg.pushManager.getSubscription();

        if (!subscription) {
          this.isSubscribed = false;
          this.isLoading = false;
          return;
        }

        await subscription.unsubscribe();

        this.isSubscribed = false;
      } catch (e) {
        console.error(e);

        this.showToast(e.message || "Something went wrong");
        console.error("Failed to unregister push subscription!");
      } finally {
        this.isLoading = false;
      }
    },
  }));
});
