const tokenKey = "playerook-token";
const userKey = "playerook-user";
const authForm = document.querySelector("#auth-form");
const listingForm = document.querySelector("#listing-form");
const authMessage = document.querySelector("#auth-message");
const listingMessage = document.querySelector("#listing-message");
const chatMessage = document.querySelector("#chat-message");
const authState = { registering: false };
const categoryLabels = { account: "Игровые аккаунты", resource: "Ресурсы и предметы", service: "Игровые услуги" };
let chatSocket;
let newsSocket;
let activeCategory = "";
let currentChatUsername = "";
let editingListing = false;

function applyTheme(theme) {
  const activeTheme = theme === "dark" ? "dark" : "light";
  document.documentElement.dataset.theme = activeTheme;
  const isDark = activeTheme === "dark";
  const button = document.querySelector("#theme-toggle");
  button.innerHTML = isDark
    ? '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20.2 15.2A8.5 8.5 0 0 1 8.8 3.8 8.5 8.5 0 1 0 20.2 15.2Z"/></svg>'
    : '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.93 4.93l1.42 1.42m11.3 11.3 1.42 1.42M2 12h2m16 0h2M4.93 19.07l1.42-1.42m11.3-11.3 1.42-1.42"/></svg>';
  button.setAttribute("aria-label", isDark ? "Тёмная тема включена. Включить светлую" : "Светлая тема включена. Включить тёмную");
  button.title = isDark ? "Тёмная тема" : "Светлая тема";
}

function currentToken() {
  return localStorage.getItem(tokenKey);
}

function currentUser() {
  try {
    return JSON.parse(localStorage.getItem(userKey) || "null");
  } catch {
    return null;
  }
}

function setMessage(element, text, isError = false) {
  element.textContent = text;
  element.classList.toggle("error", isError);
}

async function request(path, options = {}) {
  const headers = new Headers(options.headers || {});
  if (options.body) headers.set("Content-Type", "application/json");
  if (currentToken()) headers.set("Authorization", `Bearer ${currentToken()}`);
  const response = await fetch(path, { ...options, headers });
  const body = response.status === 204 ? null : await response.json();
  if (!response.ok) throw new Error(body?.error || `Ошибка запроса (${response.status})`);
  return body;
}

function renderSession() {
  const user = currentUser();
  document.querySelector("#session-state").textContent = user ? `@${user.username}` : "Гость";
  document.querySelector("#account-link").textContent = user ? "Аккаунт" : "Войти";
  document.querySelector("#account-link").href = user ? "#profile" : "#account";
  if (user) {
    document.querySelector("#profile-name").textContent = user.name;
    document.querySelector("#profile-username").textContent = `@${user.username}`;
    document.querySelector("#profile-public-id").textContent = `#${user.public_id}`;
    document.querySelector("#profile-role").textContent = user.role === "admin" ? "Администратор" : "Пользователь";
    document.querySelector("#profile-email").textContent = user.email;
    document.querySelector("#profile-created").textContent = new Date(user.created_at).toLocaleDateString("ru-KZ");
    document.querySelector("#profile-avatar").textContent = Array.from(user.name)[0]?.toUpperCase() || "P";
  }
  if (!user && ["profile", "sell"].includes(routeName())) {
    location.hash = "#account";
    return;
  }
  updateRoute();
}

async function refreshProfile() {
  if (!currentToken()) {
    renderSession();
    return;
  }
  try {
    const user = await request("/api/profile");
    localStorage.setItem(userKey, JSON.stringify(user));
  } catch (error) {
    if (error.message.includes("(401)")) {
      localStorage.removeItem(tokenKey);
      localStorage.removeItem(userKey);
    }
  }
  renderSession();
}

function routeName() {
  return (location.hash.slice(1).split(/[/?]/)[0] || "home");
}

function updateRoute() {
  const route = routeName();
  const protectedRoutes = new Set(["profile", "sell"]);
  if (protectedRoutes.has(route) && !currentUser()) {
    location.hash = "#account";
    return;
  }
  document.querySelectorAll(".page").forEach((page) => {
    page.classList.toggle("active", page.id === route);
  });
  if (route === "catalog") {
    const hashParts = location.hash.slice(1).split("/");
    activeCategory = hashParts[1] || "";
    if (!Object.hasOwn(categoryLabels, activeCategory)) activeCategory = "";
    document.querySelector("#catalog-heading").textContent = activeCategory ? categoryLabels[activeCategory] : "Все объявления";
    document.querySelector("#catalog-eyebrow").textContent = activeCategory ? "ВЫБРАННАЯ КАТЕГОРИЯ" : "PLAYEROOK MARKET";
    loadListings();
  }
  if (route === "profile" && currentUser()) loadMyListings();
  document.querySelectorAll("nav a").forEach((link) => {
    link.classList.toggle("active", link.hash.slice(1).split("/")[0] === route);
  });
  window.scrollTo({ top: 0, behavior: "smooth" });
}

function categoryName(category) {
  return ({ account: "Аккаунт", resource: "Ресурсы", service: "Услуги" })[category] || category;
}

function money(amount) {
  return new Intl.NumberFormat("ru-KZ").format(amount) + " ₸";
}

function makeListingCard(listing, own = false) {
  const card = document.createElement("article");
  card.className = "listing-card";
  const art = document.createElement("div");
  art.className = "listing-art";
  art.textContent = listing.game.slice(0, 2).toUpperCase();
  const content = document.createElement("div");
  content.className = "listing-content";
  const tag = document.createElement("span");
  tag.className = "tag";
  tag.textContent = categoryName(listing.category);
  const title = document.createElement("h3");
  title.textContent = listing.title;
  const game = document.createElement("p");
  game.className = "listing-game";
  game.textContent = listing.game;
  const description = document.createElement("p");
  description.className = "listing-description";
  description.textContent = listing.description;
  const footer = document.createElement("div");
  footer.className = "listing-footer";
  const priceSeller = document.createElement("div");
  const price = document.createElement("div");
  price.className = "price";
  price.textContent = money(listing.price_kzt);
  const seller = document.createElement("div");
  seller.className = "seller";
  seller.textContent = `@${listing.seller_username} · Player #${listing.seller_public_id}`;
  priceSeller.append(price, seller);
  const actions = document.createElement("div");
  actions.className = "listing-actions";
  if (own) {
    const edit = document.createElement("button");
    edit.className = "small-button";
    edit.type = "button";
    edit.textContent = "Изменить";
    edit.addEventListener("click", () => startEditListing(listing));
    const remove = document.createElement("button");
    remove.className = "small-button danger-button";
    remove.type = "button";
    remove.textContent = "Удалить";
    remove.addEventListener("click", () => deleteListing(listing.id));
    actions.append(edit, remove);
  } else {
    const contact = document.createElement("button");
    contact.className = "small-button";
    contact.type = "button";
    contact.textContent = `Написать @${listing.seller_username}`;
    contact.addEventListener("click", () => openChat(listing.seller_username));
    actions.append(contact);
  }
  footer.append(priceSeller, actions);
  content.append(tag, title, game, description, footer);
  card.append(art, content);
  return card;
}

async function loadListings() {
  const grid = document.querySelector("#listing-grid");
  const message = document.querySelector("#catalog-message");
  grid.replaceChildren();
  setMessage(message, "Загружаем объявления…");
  const params = new URLSearchParams({ limit: "100" });
  const query = document.querySelector("#search-input").value.trim();
  if (query) params.set("q", query);
  if (activeCategory) params.set("category", activeCategory);
  try {
    const data = await request(`/api/listings?${params}`);
    if (!data.items.length) {
      setMessage(message, "В этой категории пока нет объявлений.");
      return;
    }
    setMessage(message, "");
    data.items.forEach((listing) => grid.append(makeListingCard(listing)));
  } catch (error) {
    setMessage(message, error.message, true);
  }
}

async function loadMyListings() {
  const list = document.querySelector("#my-listings");
  list.replaceChildren();
  setMessage(document.querySelector("#profile-message"), "");
  try {
    const data = await request("/api/profile/listings");
    if (!data.items.length) {
      const empty = document.createElement("p");
      empty.className = "panel muted empty-listings";
      empty.textContent = "У тебя пока нет объявлений. Создай первое — оно появится в каталоге.";
      list.append(empty);
      return;
    }
    data.items.forEach((listing) => list.append(makeListingCard(listing, true)));
  } catch (error) {
    const message = document.createElement("p");
    message.className = "message error";
    message.textContent = error.message;
    list.append(message);
  }
}

async function loadNews(target = "#news-list") {
  const list = document.querySelector(target);
  list.replaceChildren();
  try {
    const data = await request("/api/news");
    data.items.forEach((item) => list.append(makeNewsCard(item)));
  } catch (error) {
    const message = document.createElement("p");
    message.className = "message error";
    message.textContent = error.message;
    list.append(message);
  }
}

function makeNewsCard(item) {
  const card = document.createElement("article");
  card.className = "news-card";
  const date = document.createElement("time");
  date.textContent = new Date(item.created_at).toLocaleDateString("ru-KZ");
  const title = document.createElement("h3");
  title.textContent = item.title;
  const body = document.createElement("p");
  body.textContent = item.body;
  card.append(date, title, body);
  return card;
}

function toggleAuthMode(registering) {
  authState.registering = registering;
  document.querySelector("#name-field").classList.toggle("hidden", !registering);
  document.querySelector("#username-field").classList.toggle("hidden", !registering);
  document.querySelector("#auth-heading").textContent = registering ? "Создать аккаунт" : "Войти в аккаунт";
  authForm.querySelector('input[name="password"]').autocomplete = registering ? "new-password" : "current-password";
  authForm.querySelector('input[name="name"]').required = registering;
  authForm.querySelector('input[name="username"]').required = registering;
  authForm.querySelector('button[type="submit"]').textContent = registering ? "Зарегистрироваться" : "Войти";
  document.querySelector("#register-toggle").textContent = registering ? "У меня уже есть аккаунт" : "Создать аккаунт";
}

authForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const payload = Object.fromEntries(new FormData(authForm).entries());
  const path = authState.registering ? "/api/auth/register" : "/api/auth/login";
  const registering = authState.registering;
  try {
    const data = await request(path, { method: "POST", body: JSON.stringify(payload) });
    localStorage.setItem(tokenKey, data.token);
    localStorage.setItem(userKey, JSON.stringify(data.user));
    authForm.reset();
    toggleAuthMode(false);
    renderSession();
    setMessage(authMessage, registering ? "Аккаунт создан. Добро пожаловать!" : "Вход выполнен.");
    location.hash = "#profile";
  } catch (error) {
    setMessage(authMessage, error.message, true);
  }
});

document.querySelector("#register-toggle").addEventListener("click", () => toggleAuthMode(!authState.registering));

document.querySelector("#logout-button").addEventListener("click", () => {
  localStorage.removeItem(tokenKey);
  localStorage.removeItem(userKey);
  chatSocket?.close();
  newsSocket?.close();
  renderSession();
  location.hash = "#home";
});

function resetListingForm() {
  editingListing = false;
  listingForm.reset();
  listingForm.elements.listing_id.value = "";
  document.querySelector("#listing-form-mode").textContent = "Объявление сразу появится в выбранной категории.";
  document.querySelector("#listing-submit").textContent = "Опубликовать";
  document.querySelector("#listing-cancel").classList.add("hidden");
}

function startEditListing(listing) {
  editingListing = true;
  listingForm.elements.listing_id.value = listing.id;
  listingForm.elements.title.value = listing.title;
  listingForm.elements.game.value = listing.game;
  listingForm.elements.category.value = listing.category;
  listingForm.elements.description.value = listing.description;
  listingForm.elements.price_kzt.value = listing.price_kzt;
  document.querySelector("#listing-form-mode").textContent = "Редактирование объявления";
  document.querySelector("#listing-submit").textContent = "Сохранить изменения";
  document.querySelector("#listing-cancel").classList.remove("hidden");
  location.hash = "#sell";
}

document.querySelector("#new-listing").addEventListener("click", () => {
  resetListingForm();
  location.hash = "#sell";
});
document.querySelector("#listing-cancel").addEventListener("click", resetListingForm);

listingForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const values = Object.fromEntries(new FormData(listingForm).entries());
  const listingID = values.listing_id;
  delete values.listing_id;
  values.price_kzt = Number(values.price_kzt);
  try {
    await request(editingListing ? `/api/listings/${encodeURIComponent(listingID)}` : "/api/listings", {
      method: editingListing ? "PUT" : "POST",
      body: JSON.stringify(values),
    });
    resetListingForm();
    setMessage(listingMessage, editingListing ? "Объявление обновлено." : "Объявление опубликовано.");
    location.hash = "#profile";
    loadListings();
    loadMyListings();
  } catch (error) {
    setMessage(listingMessage, error.message, true);
  }
});

async function deleteListing(id) {
  if (!confirm("Удалить это объявление?")) return;
  try {
    await request(`/api/listings/${encodeURIComponent(id)}`, { method: "DELETE" });
    await loadMyListings();
    await loadListings();
  } catch (error) {
    setMessage(document.querySelector("#profile-message"), error.message, true);
  }
}

document.querySelector("#search-form").addEventListener("submit", (event) => {
  event.preventDefault();
  loadListings();
});

document.querySelectorAll("[data-category]").forEach((button) => {
  button.addEventListener("click", () => { location.hash = `#catalog/${button.dataset.category}`; });
});
document.querySelector("#all-categories").addEventListener("click", () => { location.hash = "#catalog"; });
document.querySelector("#catalog-categories").addEventListener("click", () => { location.hash = "#categories"; });

function openChat(username) {
  location.hash = "#chats";
  document.querySelector("#chat-recipient").value = `@${username}`;
  setChatTab("private");
  connectPrivateChat();
}

function setChatTab(tab) {
  const privateTab = tab === "private";
  document.querySelector("#private-tab").classList.toggle("active", privateTab);
  document.querySelector("#channel-tab").classList.toggle("active", !privateTab);
  document.querySelector("#private-tab").setAttribute("aria-selected", String(privateTab));
  document.querySelector("#channel-tab").setAttribute("aria-selected", String(!privateTab));
  document.querySelector("#private-panel").classList.toggle("hidden", !privateTab);
  document.querySelector("#channel-panel").classList.toggle("hidden", privateTab);
  if (privateTab) {
    newsSocket?.close();
  } else {
    chatSocket?.close();
    connectNewsChannel();
  }
}

document.querySelector("#private-tab").addEventListener("click", () => setChatTab("private"));
document.querySelector("#channel-tab").addEventListener("click", () => setChatTab("news"));
document.querySelector("#chat-start-form").addEventListener("submit", (event) => {
  event.preventDefault();
  connectPrivateChat();
});

function connectPrivateChat() {
  const recipient = document.querySelector("#chat-recipient").value.trim().replace(/^@/, "").toLowerCase();
  if (!currentToken()) {
    setMessage(chatMessage, "Войди в аккаунт, чтобы пользоваться чатом.", true);
    location.hash = "#account";
    return;
  }
  if (!/^[a-z0-9_]{3,24}$/.test(recipient)) {
    setMessage(chatMessage, "Никнейм: 3–24 символа, латиница, цифры и _.", true);
    return;
  }
  currentChatUsername = recipient;
  chatSocket?.close();
  document.querySelector("#chat-title").textContent = `Диалог с @${recipient}`;
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  chatSocket = new WebSocket(`${scheme}//${location.host}/ws/chat?with=${encodeURIComponent(recipient)}&token=${encodeURIComponent(currentToken())}`);
  chatSocket.addEventListener("open", () => setMessage(chatMessage, "Чат подключён."));
  chatSocket.addEventListener("message", (event) => {
    const data = JSON.parse(event.data);
    if (data.type === "history") {
      document.querySelector("#chat-messages").replaceChildren();
      data.items.forEach(appendChatMessage);
    } else if (data.type === "message") {
      appendChatMessage(data.item);
    } else if (data.error) {
      setMessage(chatMessage, data.error, true);
    }
  });
  chatSocket.addEventListener("error", () => setMessage(chatMessage, "Не удалось подключиться к чату.", true));
}

function appendChatMessage(item) {
  const list = document.querySelector("#chat-messages");
  const bubble = document.createElement("div");
  bubble.className = "chat-bubble";
  if (item.sender_username === currentUser()?.username) bubble.classList.add("own");
  const sender = document.createElement("strong");
  sender.textContent = `@${item.sender_username || "игрок"} `;
  const text = document.createElement("span");
  text.textContent = item.text;
  const time = document.createElement("small");
  time.textContent = new Date(item.created_at).toLocaleTimeString("ru-KZ", { hour: "2-digit", minute: "2-digit" });
  bubble.append(sender, text, time);
  list.append(bubble);
  list.scrollTop = list.scrollHeight;
}

document.querySelector("#chat-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const textInput = document.querySelector("#chat-input");
  if (!chatSocket || chatSocket.readyState !== WebSocket.OPEN) {
    if (!currentChatUsername) {
      setMessage(chatMessage, "Сначала выбери собеседника по никнейму.", true);
      return;
    }
    connectPrivateChat();
    setMessage(chatMessage, "Подключаемся — отправь сообщение ещё раз.");
    return;
  }
  chatSocket.send(JSON.stringify({ recipient_username: currentChatUsername, text: textInput.value.trim() }));
  textInput.value = "";
});

function connectNewsChannel() {
  const list = document.querySelector("#chat-news-list");
  if (!currentToken()) {
    setMessage(document.querySelector("#channel-message"), "Войди, чтобы подключиться к каналу новостей.", true);
    return;
  }
  if (newsSocket?.readyState === WebSocket.OPEN || newsSocket?.readyState === WebSocket.CONNECTING) return;
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  newsSocket = new WebSocket(`${scheme}//${location.host}/ws/chat?channel=news&token=${encodeURIComponent(currentToken())}`);
  newsSocket.addEventListener("open", () => setMessage(document.querySelector("#channel-message"), "Подключено к каналу новостей."));
  newsSocket.addEventListener("message", (event) => {
    const data = JSON.parse(event.data);
    if (data.type === "history") {
      list.replaceChildren();
      data.items.forEach((item) => list.append(makeNewsCard(item)));
    } else if (data.type === "news") {
      list.prepend(makeNewsCard(data.item));
    } else if (data.error) {
      setMessage(document.querySelector("#channel-message"), data.error, true);
    }
  });
  newsSocket.addEventListener("error", () => setMessage(document.querySelector("#channel-message"), "Не удалось подключиться к новостям.", true));
}

document.querySelector("#theme-toggle").addEventListener("click", () => {
  const nextTheme = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
  localStorage.setItem("playerook-theme", nextTheme);
  applyTheme(nextTheme);
});

window.addEventListener("hashchange", updateRoute);
toggleAuthMode(false);
applyTheme(localStorage.getItem("playerook-theme"));
renderSession();
refreshProfile();
loadNews();
