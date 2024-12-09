/**
 *  This function fetches a random dad joke and submits a push message via the /api/v1/push endpoint.
 *  Only needed for demo purposes. Do not use in production.
 *
 *  @param {Request} req
 *  @returns {Promise<Response>}
 *  */
export default async (req) => {
  if (req.method !== "PUT") {
    return new Response("Method Not Allowed", {
      headers: {
        "content-type": "text/plain",
        allow: "PUT",
      },
      status: 405,
    });
  }

  const auth = req.headers.get("authorization");
  const [, encoded] = auth?.split("Basic ") ?? [];

  if (!encoded) {
    return new Response("Unauthorized", {
      status: 401,
      headers: {
        "content-type": "text/plain",
        "www-authenticate": 'Basic realm="Dad Joke Demo"',
      },
    });
  }

  const [username, password] = atob(encoded).split(":");

  if (
    username !== "demo" ||
    password !== Netlify.env.get("BASIC_AUTH_PASSWORD")
  ) {
    return new Response("Forbidden", {
      status: 403,
      headers: {
        "content-type": "text/plain",
      },
    });
  }

  const icanhazdadjoke = new URL("https://icanhazdadjoke.com");
  const headers = new Headers({
    accept: "application/json",
    "user-agent": `go-web-push-server (${Netlify.env.get("DEPLOY_URL")})`,
  });

  const res = await fetch(icanhazdadjoke, {
    headers,
  });

  if (!res.ok) {
    return new Response("Failed to fetch joke.", {
      status: res.status,
    });
  }

  const joke = await res.json();

  const params = new URLSearchParams({
    // push message will be retained for one day before being dropped
    ttl: 60 * 60 * 24,
    // topic is dad joke
    topic: "dad-joke",
  });

  const url = new URL("/api/v1/push", req.url);
  url.search = params.toString();

  const body = {
    title: "Today's Dad Joke",
    body: joke.joke,
    url: "https://icanhazdadjoke.com/j/" + joke.id,
    icon: "/assets/icon.png",
  };

  return fetch(url, {
    method: "POST",
    body: JSON.stringify(body),
    headers: {
      "content-type": "application/json",
      authorization: `Basic ${btoa(
        `demo:${Netlify.env.get("BASIC_AUTH_PASSWORD")}`
      )}`,
    },
  });
};

export const config = {
  path: "/demo/dad-joke",
};
