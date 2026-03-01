async function api(path, opts = {}) {
  const res = await fetch(path, {
    ...opts,
    headers: {
      "Content-Type": "application/json",
      ...(opts.headers || {}),
    },
    cache: "no-store",
  });
  return res;
}

function qs(id) {
  return document.getElementById(id);
}

function show(el) {
  el.classList.remove("hidden");
}
function hide(el) {
  el.classList.add("hidden");
}

async function login(password) {
  const res = await api("/login", {
    method: "POST",
    body: JSON.stringify({ password }),
  });
  if (res.status === 204) return true;
  return false;
}

async function logout() {
  await api("/logout", { method: "POST" });
}

async function loadItems() {
  const res = await api("/api/items");
  if (res.status === 401) return { unauthorized: true };
  if (!res.ok) throw new Error(`items: ${res.status}`);
  return { unauthorized: false, items: await res.json() };
}

async function loadHistory() {
  const res = await api("/api/history?limit=20");
  if (res.status === 401) return { unauthorized: true };
  if (!res.ok) throw new Error(`history: ${res.status}`);
  return { unauthorized: false, history: await res.json() };
}

async function addItem(text) {
  const res = await api("/api/items", {
    method: "POST",
    body: JSON.stringify({ text }),
  });
  if (!res.ok) throw new Error(`add: ${res.status}`);
  return await res.json();
}

async function toggleItem(id) {
  const res = await api(`/api/items/${id}/toggle`, { method: "POST" });
  if (!res.ok) throw new Error(`toggle: ${res.status}`);
  return await res.json();
}

async function deleteItem(id) {
  const res = await api(`/api/items/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error(`delete: ${res.status}`);
}

async function clearDone() {
  const res = await api("/api/items/clear-done", { method: "POST" });
  if (!res.ok) throw new Error(`clear: ${res.status}`);
}

function renderItems(items) {
  const ul = qs("items");
  ul.innerHTML = "";

  for (const it of items) {
    const li = document.createElement("li");
    li.className = "li";

    const cb = document.createElement("input");
    cb.type = "checkbox";
    cb.checked = !!it.done;
    cb.addEventListener("change", async () => {
      try {
        await toggleItem(it.id);
        await refresh();
      } catch (e) {
        console.error(e);
        cb.checked = !cb.checked;
      }
    });

    const txt = document.createElement("span");
    txt.textContent = it.text;
    txt.className = it.done ? "done" : "";

    const del = document.createElement("button");
    del.type = "button";
    del.className = "ghost";
    del.textContent = "×";
    del.title = "Remove";
    del.addEventListener("click", async () => {
      try {
        await deleteItem(it.id);
        await refresh();
      } catch (e) {
        console.error(e);
      }
    });

    li.appendChild(cb);
    li.appendChild(txt);
    li.appendChild(del);
    ul.appendChild(li);
  }
}

function renderHistory(history) {
  const ul = qs("history");
  ul.innerHTML = "";

  for (const t of history) {
    const li = document.createElement("li");
    li.className = "li";

    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "linkish";
    btn.textContent = t.text;
    btn.addEventListener("click", async () => {
      try {
        await addItem(t.text);
        await refresh();
      } catch (e) {
        console.error(e);
      }
    });

    li.appendChild(btn);
    ul.appendChild(li);
  }
}

async function refresh() {
  const a = await loadItems();
  if (a.unauthorized) {
    show(qs("loginCard"));
    hide(qs("appCard"));
    hide(qs("btnLogout"));
    return;
  }

  const b = await loadHistory();
  if (b.unauthorized) {
    show(qs("loginCard"));
    hide(qs("appCard"));
    hide(qs("btnLogout"));
    return;
  }

  hide(qs("loginCard"));
  show(qs("appCard"));
  show(qs("btnLogout"));

  renderItems(a.items);
  renderHistory(b.history);
}

function registerServiceWorker() {
  if (!("serviceWorker" in navigator)) return;
  navigator.serviceWorker.register("/sw.js").catch(() => {});
}

document.addEventListener("DOMContentLoaded", () => {
  registerServiceWorker();

  qs("loginForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const pw = qs("pw").value;

    const ok = await login(pw);
    if (!ok) {
      qs("loginErr").textContent = "Login failed.";
      show(qs("loginErr"));
      return;
    }
    hide(qs("loginErr"));
    qs("pw").value = "";
    await refresh();
  });

  qs("btnLogout").addEventListener("click", async () => {
    await logout();
    await refresh();
  });

  qs("addForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const inp = qs("newText");
    const text = inp.value.trim();
    if (!text) return;

    try {
      await addItem(text);
      inp.value = "";
      await refresh();
    } catch (e2) {
      console.error(e2);
    }
  });

  qs("btnClearDone").addEventListener("click", async () => {
    try {
      await clearDone();
      await refresh();
    } catch (e) {
      console.error(e);
    }
  });

  refresh().catch(console.error);
});
