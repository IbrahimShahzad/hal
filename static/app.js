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
    ts.textContent = new Date(u.timestamp).toLocaleString();
    wrap.appendChild(ts);

    const msg = document.createElement("div");
    msg.className = "msg";
    wrap.appendChild(msg);

    if (u.tags && u.tags.length) {
        const t = document.createElement("div");
        t.className = "tags";
        t.textContent = "tags: " + u.tags.join(", ");
        wrap.appendChild(t);
    }

    return wrap;
}

async function animateNewEntry(u) {
    const container = document.getElementById("log");
    const entry = createEntrySkeleton(u);

    const msgEl = entry.querySelector(".msg");

    // Add cursor
    const cursor = document.createElement("span");
    cursor.className = "cursor";
    msgEl.appendChild(cursor);

    container.prepend(entry);

    // typewriter effect
    const fullText = u.message;
    msgEl.removeChild(cursor);
    await typewriter(msgEl, fullText, 18);

    // Add cursor back at the end
    msgEl.appendChild(cursor);

    // After 1.5s remove cursor permanently
    setTimeout(() => cursor.remove(), 1500);
}

async function loadInitial() {
    const res = await fetch("/initial");
    const list = await res.json();
    const container = document.getElementById("log");

    for (let i = list.length - 1; i >= 0; i--) {
        const e = createEntrySkeleton(list[i]);
        e.querySelector(".msg").textContent = list[i].message;
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

loadInitial();
startStream();
