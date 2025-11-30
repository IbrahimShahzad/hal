// Get current user from URL path
function getCurrentUser() {
    const path = window.location.pathname;
    if (path === '/') return null;
    if (path.startsWith('/user/')) {
        return path.substring(6); // Remove "/user/" prefix
    }
    return null;
}

async function typewriter(el, text, speed = 25) {
    el.textContent = "";
    for (let i = 0; i < text.length; i++) {
        el.textContent += text[i];
        await new Promise(r => setTimeout(r, speed));
    }
}

function createEntrySkeleton(u) {
    const wrap = document.createElement("div");
    wrap.className = "entry";

    const ts = document.createElement("div");
    ts.className = "ts";
    wrap.appendChild(ts);

    const user = document.createElement("div");
    user.className = "username";
    wrap.appendChild(user);

    const msg = document.createElement("div");
    msg.className = "msg";
    wrap.appendChild(msg);

    if (u.tags && u.tags.length) {
        const t = document.createElement("div");
        t.className = "tags";
        wrap.appendChild(t);
    }

    return wrap;
}

async function animateNewEntry(u) {
    const currentUser = getCurrentUser();
    
    if (currentUser && u.username !== currentUser) {
        return;
    }
    
    const container = document.getElementById("log");
    const entry = createEntrySkeleton(u);

    const tsEl = entry.querySelector(".ts");
    const userEl = entry.querySelector(".username");
    const msgEl = entry.querySelector(".msg");
    const tagsEl = entry.querySelector(".tags");

    container.prepend(entry);

    const timestampText = new Date(u.timestamp).toLocaleString();
    await typewriter(tsEl, timestampText, 35);

    if (userEl && u.username && !currentUser) {
        const usernameText = `[${u.username}]`;
        await typewriter(userEl, usernameText, 25);
    } else if (userEl) {
        userEl.remove();
    }

    const cursor = document.createElement("span");
    cursor.className = "cursor";
    msgEl.appendChild(cursor);

    const fullText = u.message;
    msgEl.removeChild(cursor);
    await typewriter(msgEl, fullText, 75);

    msgEl.appendChild(cursor);

    if (tagsEl && u.tags && u.tags.length) {
        const tagsText = "tags: " + u.tags.join(", ");
        await typewriter(tagsEl, tagsText, 35);
    }
    setTimeout(() => cursor.remove(), 1500);
}

async function loadInitial() {
    const currentUser = getCurrentUser();
    const endpoint = currentUser ? `/initial/${currentUser}` : '/initial';
    
    console.log('Current user:', currentUser);
    console.log('Fetching from:', endpoint);
    
    const res = await fetch(endpoint);
    const list = await res.json();
    
    console.log('Received data:', list);
    
    const container = document.getElementById("log");

    for (let i = list.length - 1; i >= 0; i--) {
        const e = createEntrySkeleton(list[i]);
        
        const tsEl = e.querySelector(".ts");
        tsEl.textContent = new Date(list[i].timestamp).toLocaleString();
        
        const userEl = e.querySelector(".username");
        if (userEl && list[i].username && !currentUser) {
            userEl.textContent = `[${list[i].username}]`;
        } else if (userEl) {
            userEl.remove();
        }
        e.querySelector(".msg").textContent = list[i].message;
        const tagsEl = e.querySelector(".tags");
        if (tagsEl && list[i].tags && list[i].tags.length) {
            tagsEl.textContent = "tags: " + list[i].tags.join(", ");
        }
        
        container.append(e);
    }
}

function startStream() {
    const es = new EventSource("/stream");
    es.onmessage = async e => {
        try {
            const u = JSON.parse(e.data);
            await animateNewEntry(u);
        } catch (err) {
            console.error(err);
        }
    };
}

// Update page title and header based on current user
function updatePageTitle() {
    const currentUser = getCurrentUser();
    if (currentUser) {
        document.title = `HAL Activity Monitor - ${currentUser}`;
        const header = document.querySelector('h2');
        if (header) {
            header.textContent = `[ HAL ] Monitoring Protocol: ${currentUser.toUpperCase()}`;
        }
    }
}

// Initialize page
updatePageTitle();
loadInitial();
startStream();
