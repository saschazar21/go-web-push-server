importScripts(
  "https://cdn.jsdelivr.net/npm/workbox-sw@7.3.0/build/workbox-sw.min.js"
);

self.addEventListener("notificationclick", (event) => {
  event.notification.close();

  event.waitUntil(
    clients.matchAll({ type: "window" }).then((clientList) => {
      for (let i = 0; i < clientList.length; i++) {
        const client = clientList[i];
        if ("focus" in client) {
          return client.focus();
        }
      }

      if (clients.openWindow) {
        return clients.openWindow("/");
      }
    })
  );
});

// inspired by: https://github.com/GoogleChrome/samples/blob/gh-pages/push-messaging-and-notifications/service-worker.js
self.addEventListener("push", (event) => {
  let data;
  try {
    data = {
      title: "New push message",
      icon: "https://raw.githubusercontent.com/twitter/twemoji/master/assets/72x72/1f3c4.png",
      tag: "custom",
      ...event.data.json(),
    };
  } catch (_e) {
    data = {
      title: "New push message",
      body: event.data.text(),
      icon: "https://raw.githubusercontent.com/twitter/twemoji/master/assets/72x72/1f3ca.png",
      tag: "default",
    };
  }

  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: data.icon,
      tag: data.tag,
    })
  );
});

self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "SKIP_WAITING") {
    self.skipWaiting();
  }
});

workbox.precaching.precacheAndRoute(self.__WB_MANIFEST);
