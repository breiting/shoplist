async function api(path, opts = {}) {
  const res = await fetch(path, {
    ...opts,
    headers: {
      ...(opts.headers || {}),
      "Content-Type":
        (opts.headers && opts.headers["Content-Type"]) || "application/json",
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
  return res.status === 204;
}

async function logout() {
  await api("/logout", { method: "POST" });
}

async function loadConfig() {
  const res = await api("/api/config");
  if (res.status === 401) return { unauthorized: true };
  if (!res.ok) throw new Error(`config: ${res.status}`);
  return { unauthorized: false, config: await res.json() };
}

async function loadItems(shop) {
  const res = await api(`/api/items?shop=${encodeURIComponent(shop)}`);
  if (res.status === 401) return { unauthorized: true };
  if (!res.ok) throw new Error(`items: ${res.status}`);
  return { unauthorized: false, items: await res.json() };
}

async function loadHistory(shop) {
  const res = await api(
    `/api/history?shop=${encodeURIComponent(shop)}&limit=20`,
  );
  if (res.status === 401) return { unauthorized: true };
  if (!res.ok) throw new Error(`history: ${res.status}`);
  return { unauthorized: false, history: await res.json() };
}

async function addItem(shop, text) {
  const res = await api("/api/items", {
    method: "POST",
    body: JSON.stringify({ shop, text }),
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

async function clearDone(shop) {
  const res = await api(
    `/api/items/clear-done?shop=${encodeURIComponent(shop)}`,
    {
      method: "POST",
    },
  );
  if (!res.ok) throw new Error(`clear: ${res.status}`);
}

async function setQty(id, qty) {
  const res = await api(`/api/items/${id}/qty`, {
    method: "POST",
    body: JSON.stringify({ qty }),
  });
  if (!res.ok) throw new Error(`qty: ${res.status}`);
  return await res.json();
}

function renderItems(items) {
  const ul = qs("items");
  ul.innerHTML = "";

  // Be defensive: backend should return [], but never trust the network.
  if (!Array.isArray(items) || items.length === 0) {
    const li = document.createElement("li");
    li.className = "liEmpty";
    li.textContent = "No items.";
    ul.appendChild(li);
    return;
  }

  for (const it of items) {
    const li = document.createElement("li");
    li.className = "li";

    // Checkbox
    const cbWrap = document.createElement("label");
    cbWrap.className = "cb";
    cbWrap.dataset.checked = it.done ? "1" : "0";

    const cb = document.createElement("input");
    cb.type = "checkbox";
    cb.checked = !!it.done;

    const mark = document.createElement("span");
    mark.className = "mark";

    cb.addEventListener("change", async () => {
      cbWrap.dataset.checked = cb.checked ? "1" : "0";
      try {
        await toggleItem(it.id);
        await refresh();
      } catch (e) {
        console.error(e);
        cb.checked = !cb.checked;
        cbWrap.dataset.checked = cb.checked ? "1" : "0";
      }
    });

    cbWrap.appendChild(cb);
    cbWrap.appendChild(mark);

    // Text
    const txt = document.createElement("span");
    txt.textContent = it.text;
    txt.className = "itemText" + (it.done ? " done" : "");

    // Qty pill (tap to edit)
    const qtyHost = document.createElement("span");
    qtyHost.className = "qtyHost";

    const qtyBtn = document.createElement("button");
    qtyBtn.type = "button";
    qtyBtn.className = "qtyPill" + (it.qty && it.qty.trim() ? "" : " empty");
    qtyBtn.textContent = it.qty && it.qty.trim() ? it.qty.trim() : "Qty";
    qtyBtn.title = "Set quantity";

    qtyBtn.addEventListener("click", () =>
      startQtyEdit(qtyHost, it.id, it.qty || ""),
    );

    qtyHost.appendChild(qtyBtn);

    // Delete
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

    li.appendChild(cbWrap);
    li.appendChild(txt);
    li.appendChild(qtyHost);
    li.appendChild(del);
    ul.appendChild(li);
  }
}

function startQtyEdit(host, id, currentQty) {
  // Close any other active editor
  const prev = document.querySelector(".qtyInput");
  if (prev) {
    prev.blur(); // triggers its save handler
  }

  host.innerHTML = "";

  const inp = document.createElement("input");
  inp.type = "text";
  inp.inputMode = "text";
  inp.autocomplete = "off";
  inp.className = "qtyInput";
  inp.placeholder = "Qty";
  inp.value = currentQty || "";

  let saved = false;
  const commit = async () => {
    if (saved) return;
    saved = true;
    try {
      await setQty(id, inp.value.trim());
      await refresh();
    } catch (e) {
      console.error(e);
      await refresh();
    }
  };

  inp.addEventListener("keydown", async (ev) => {
    if (ev.key === "Enter") {
      ev.preventDefault();
      inp.blur(); // will commit via blur
    } else if (ev.key === "Escape") {
      ev.preventDefault();
      saved = true; // prevent commit
      refresh().catch(console.error);
    }
  });

  inp.addEventListener("blur", () => {
    commit().catch(console.error);
  });

  host.appendChild(inp);
  inp.focus();
  inp.select();
}

function renderHistory(history) {
  const ul = qs("history");
  ul.innerHTML = "";

  for (const t of history) {
    const li = document.createElement("li");
    li.className = "liHistory";

    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "linkish";
    btn.textContent = t.text;

    btn.addEventListener("click", async () => {
      try {
        const shop = qs("shopSelect").value;
        await addItem(shop, t.text);
        await refresh();
      } catch (e) {
        console.error(e);
      }
    });

    li.appendChild(btn);
    ul.appendChild(li);
  }
}

function ensureShopSelect(cfg) {
  const sel = qs("shopSelect");
  if (!sel) return cfg.defaultShop;

  if (sel.dataset.ready !== "1") {
    sel.innerHTML = "";
    for (const s of cfg.shops || []) {
      const opt = document.createElement("option");
      opt.value = s;
      opt.textContent = s;
      sel.appendChild(opt);
    }
    sel.dataset.ready = "1";

    sel.addEventListener("change", async () => {
      localStorage.setItem("shoplist_shop", sel.value);
      await refresh();
    });
  }

  const saved = localStorage.getItem("shoplist_shop");
  const preferred =
    saved && (cfg.shops || []).includes(saved) ? saved : cfg.defaultShop;
  if (preferred && sel.value !== preferred) {
    sel.value = preferred;
  }

  return sel.value || cfg.defaultShop;
}

async function refresh() {
  const c = await loadConfig();
  if (c.unauthorized) {
    show(qs("loginCard"));
    hide(qs("appCard"));
    hide(qs("btnLogout"));
    return;
  }

  const shop = ensureShopSelect(c.config);

  // Load both in parallel; never skip history just because items are empty
  const [a, b] = await Promise.all([loadItems(shop), loadHistory(shop)]);

  if (a.unauthorized || b.unauthorized) {
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

// iOS: tap outside inputs/selects to close keyboard
const blurIfNeeded = (ev) => {
  const ae = document.activeElement;
  if (!ae) return;

  const tag = ae.tagName;
  const isEditable = tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
  if (!isEditable) return;

  const t = ev.target;
  if (!t) return;

  const ttag = t.tagName;
  const isTargetEditable =
    ttag === "INPUT" || ttag === "TEXTAREA" || ttag === "SELECT";
  if (isTargetEditable) return;

  ae.blur();
};

document.addEventListener("pointerdown", blurIfNeeded, { passive: true });
document.addEventListener("touchstart", blurIfNeeded, { passive: true });

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

  qs("btnRefresh")?.addEventListener("click", async () => {
    try {
      await refresh();
    } catch (e) {
      console.error(e);
    }
  });

  qs("addForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    const inp = qs("newText");
    const text = inp.value.trim();
    if (!text) return;

    const shop = qs("shopSelect") ? qs("shopSelect").value : "";

    try {
      await addItem(shop, text);
      inp.value = "";
      inp.focus();
      await refresh();
    } catch (e2) {
      console.error(e2);
    }
  });

  qs("btnClearDone").addEventListener("click", async () => {
    try {
      const shop = qs("shopSelect") ? qs("shopSelect").value : "";
      await clearDone(shop);
      await refresh();
    } catch (e) {
      console.error(e);
    }
  });

  // Pull-to-refresh (cleaner trigger)
  (function setupPullToRefresh() {
    const ptr = qs("ptr");
    const ptrText = qs("ptrText");
    if (!ptr || !ptrText) return;

    let startY = 0;
    let pulling = false;
    let armed = false;
    let visible = false;

    const showEl = (el) => el.classList.remove("hidden");
    const hideEl = (el) => el.classList.add("hidden");

    const thresholdShow = 20; // when to first show indicator
    const thresholdRefresh = 60; // when refresh becomes armed

    window.addEventListener(
      "touchstart",
      (e) => {
        if (e.touches.length !== 1) return;
        if (window.scrollY !== 0) return;

        startY = e.touches[0].clientY;
        pulling = true;
        armed = false;
        visible = false;
      },
      { passive: true },
    );

    window.addEventListener(
      "touchmove",
      (e) => {
        if (!pulling) return;

        const y = e.touches[0].clientY;
        const dy = y - startY;

        if (dy <= 0) return;

        // Only show indicator after small pull
        if (!visible && dy > thresholdShow) {
          ptrText.textContent = "Pull to refresh";
          showEl(ptr);
          visible = true;
        }

        if (!visible) return;

        if (dy > thresholdRefresh) {
          ptrText.textContent = "Release to refresh";
          armed = true;
        } else {
          ptrText.textContent = "Pull to refresh";
          armed = false;
        }
      },
      { passive: true },
    );

    window.addEventListener(
      "touchend",
      async () => {
        if (!pulling) return;

        pulling = false;

        if (armed) {
          ptrText.textContent = "Refreshing…";
          try {
            await refresh();
          } catch (e) {
            console.error(e);
          }
        }

        if (visible) hideEl(ptr);
        armed = false;
        visible = false;
      },
      { passive: true },
    );

    window.addEventListener("touchcancel", () => {
      pulling = false;
      armed = false;
      visible = false;
      hideEl(ptr);
    });
  })();

  refresh().catch(console.error);
});
