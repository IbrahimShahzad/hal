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
    const container = document.getElementById("log");
    const entry = createEntrySkeleton(u);

    const tsEl = entry.querySelector(".ts");
    const msgEl = entry.querySelector(".msg");
    const tagsEl = entry.querySelector(".tags");

    container.prepend(entry);

    const timestampText = new Date(u.timestamp).toLocaleString();
    await typewriter(tsEl, timestampText, 35);

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
    const res = await fetch("/initial");
    const list = await res.json();
    const container = document.getElementById("log");

    for (let i = list.length - 1; i >= 0; i--) {
        const e = createEntrySkeleton(list[i]);
        
        // Set timestamp
        const tsEl = e.querySelector(".ts");
        tsEl.textContent = new Date(list[i].timestamp).toLocaleString();
        
        // Set message
        e.querySelector(".msg").textContent = list[i].message;
        
        // Set tags if they exist
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

loadInitial();
startStream();
