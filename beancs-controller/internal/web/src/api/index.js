const API = "/v1/api";
import { claimFromJwt, browserRedirectURI } from "../utils/index";
export function makeAPI(token, onUnauthorized) {
  async function request(path, options = {}) {
    const res = await fetch(API + path, {
      ...options,
      headers: {
        ...(options.body ? {"Content-Type": "application/json"} : {}),
        Authorization: `Bearer ${token}`,
        ...(options.headers || {}),
      },
    });
    const data = await res.json().catch(() => ({}));
    if (res.status === 401 && isSessionAuthError(data)) onUnauthorized();
    if (!res.ok) throw new Error(data.error || data.error_description || `Request failed (${res.status})`);
    return data;
  }
  async function stream(path, options = {}) {
    const res = await fetch(API + path, {
      ...options,
      headers: {
        Authorization: `Bearer ${token}`,
        ...(options.headers || {}),
      },
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      if (res.status === 401 && isSessionAuthError(data)) onUnauthorized();
      throw new Error(data.error || data.error_description || `Request failed (${res.status})`);
    }
    return res;
  }
  return {
    get: (path) => request(path),
    post: (path, body) => request(path, {method: "POST", body: JSON.stringify(body)}),
    put: (path, body) => request(path, {method: "PUT", body: JSON.stringify(body)}),
    patch: (path, body) => request(path, {method: "PATCH", body: JSON.stringify(body)}),
    delete: (path) => request(path, {method: "DELETE"}),
    stream,
  };
}

export function isSessionAuthError(data) {
  const error = String(data?.error || data?.error_description || "").toLowerCase();
  return error === "missing token" || error === "invalid token" || error === "invalid api key";
}

export async function consumeTextStream(res, onChunk) {
  const reader = res.body?.getReader();
  if (!reader) throw new Error("Streaming logs are not supported by this browser.");
  const decoder = new TextDecoder();
  while (true) {
    const {value, done} = await reader.read();
    if (done) break;
    onChunk(decoder.decode(value, {stream: true}));
  }
  const remaining = decoder.decode();
  if (remaining) onChunk(remaining);
}

export async function publicJSON(path, options = {}) {
  const res = await fetch(path, options);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || "Request failed");
  return data;
}

export async function finishLogin(config) {
  const params = new URLSearchParams(location.search);
  const code = params.get("code");
  const returnedState = params.get("state");
  const expectedState = sessionStorage.getItem("beancs.oauthState");
  const verifier = sessionStorage.getItem("beancs.pkceVerifier");
  const expectedNonce = sessionStorage.getItem("beancs.oauthNonce");
  if (!code || !verifier || returnedState !== expectedState) throw new Error("Login callback was incomplete.");
  const data = await publicJSON(`${API}/ui/oauth/token`, {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify({code, redirect_uri: browserRedirectURI(), code_verifier: verifier}),
  });
  if (data.id_token && claimFromJwt(data.id_token, "nonce") !== expectedNonce) {
    throw new Error("Login callback returned an invalid id_token nonce.");
  }

  sessionStorage.removeItem("beancs.pkceVerifier");
  sessionStorage.removeItem("beancs.oauthState");
  sessionStorage.removeItem("beancs.oauthNonce");
  return data.access_token;
}