/**
 *  This function is a proxy to the /api/v1/subscribe endpoint, with added authorization credentials.
 *  Only needed for testing purposes. Do not use in production.
 *
 *  @param {Request} req
 *  @returns {Promise<Response>}
 *  */
export default async (req) => {
  if (req.method !== "POST") {
    return new Response("Method Not Allowed", {
      headers: {
        "content-type": "text/plain",
        allow: "POST",
      },
      status: 405,
    });
  }

  const url = new URL("/api/v1/subscribe", req.url);
  let body;

  try {
    body = await req.json();
  } catch (e) {
    console.error(e);

    return new Response("Bad Request", {
      headers: {
        "content-type": "text/plain",
      },
      status: 400,
    });
  }

  const subscription = {
    clientId: "demo",
    subscription: body,
  };

  const headers = {
    "content-type": "application/json",
    authorization: `Basic ${btoa(`demo:${process.env.BASIC_AUTH_PASSWORD}`)}`,
  };

  const res = await fetch(url, {
    body: JSON.stringify(subscription),
    method: "POST",
    headers,
  });

  if (res.ok) {
    const url = new URL("/demo/dad-joke", req.url);

    await fetch(url, {
      method: "PUT",
      headers: {
        ...headers,
        "content-type": "text/plain",
      },
    });
  }

  return res;
};

export const config = {
  runtime: "edge",
};
