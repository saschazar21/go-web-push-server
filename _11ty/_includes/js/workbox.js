importScripts(
  "https://cdn.jsdelivr.net/npm/workbox-sw@7.3.0/build/workbox-sw.min.js",
);

function handleNotificationClick(event) {
  event.notification.close();

  const targetUrl = event.notification.data?.url || "/";

  event.waitUntil(
    clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then((clientList) => {
        // Try to focus an existing tab with the same URL
        for (const client of clientList) {
          if (client.url === targetUrl && "focus" in client) {
            return client.focus();
          }
        }
        return clients.openWindow(targetUrl);
      }),
  );
}

self.addEventListener("notificationclick", handleNotificationClick);

// inspired by: https://github.com/GoogleChrome/samples/blob/gh-pages/push-messaging-and-notifications/service-worker.js
function handlePushEvent(event) {
  let data;

  if (event.data) {
    try {
      data = { title: "New push message", ...event.data.json() };
    } catch (_e) {
      data = { title: "New push message", body: event.data.text() };
    }
  } else {
    data = { title: "New notification", body: "" };
  }

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

  const showNotification = self.registration.showNotification(data.title, {
    body: data.body,
    icon: data.icon,
    tag: data.tag,
    renotify: data.renotify ?? true,
    data: { url: data.url },
  });

  event.waitUntil(showNotification);
}

self.addEventListener("push", handlePushEvent);

function handleMessageEvent(event) {
  if (event.data && event.data.type === "SKIP_WAITING") {
    self.skipWaiting();
  }
}

self.addEventListener("message", handleMessageEvent);

workbox.routing.registerRoute(
  /^https:\/\/cdn\.jsdelivr\.net\/npm\/.+$/,
  new workbox.strategies.CacheFirst({
    cacheName: "jsdelivr-cache",
    plugins: [
      new workbox.expiration.ExpirationPlugin({
        maxEntries: 30,
      }),
    ],
  }),
);

workbox.precaching.precacheAndRoute(self.__WB_MANIFEST);
