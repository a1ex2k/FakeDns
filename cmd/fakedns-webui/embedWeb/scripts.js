(() => {
    const msgEl = document.getElementById("msg");
    const countEl = document.getElementById("count");
    const listEl = document.getElementById("list");
    const loadingEl = document.getElementById("loading");
    const emptyEl = document.getElementById("empty");
    const clearBtn = document.getElementById("clearBtn");
    const domainsInput = document.getElementById("domainsInput");
    const searchInput = document.getElementById("searchInput");
    const clearSearchBtn = document.getElementById("clearSearchBtn");

    let allDomains = [];
    let currentQuery = "";

    function decodePlus(s) {
        // /ui/?msg=Done+ok
        return decodeURIComponent((s || "").replace(/\+/g, " "));
    }

    function showMessage(text) {
        if (!text) return;
        msgEl.textContent = decodePlus(text);
        msgEl.hidden = false;
    }

    function setLoading(v) {
        loadingEl.style.display = v ? "block" : "none";
    }

    function filterDomains(domains, q) {
        const query = (q || "").trim().toLowerCase();
        if (!query) return domains;
        return domains.filter(d => d.toLowerCase().includes(query));
    }

    function applyFilterAndRender() {
        const filtered = filterDomains(allDomains, currentQuery);
        render(filtered);
    }

    function render(domains) {
        listEl.innerHTML = "";
        countEl.textContent = String(domains.length);

        if (!domains.length) {
            emptyEl.hidden = false;
            return;
        }
        emptyEl.hidden = true;

        for (const d of domains) {
            const li = document.createElement("li");
            li.className = "item";

            const left = document.createElement("div");
            left.className = "domain";
            left.textContent = d;

            const right = document.createElement("div");
            right.className = "actions";

            // ВАЖНО: делаем обычный form POST /delete, чтобы серверный redirect работал как раньше
            const form = document.createElement("form");
            form.method = "post";
            form.action = "/delete";

            const inp = document.createElement("input");
            inp.type = "hidden";
            inp.name = "domain";
            inp.value = d;

            const btn = document.createElement("button");
            btn.type = "submit";
            btn.className = "btn danger";
            btn.textContent = "Delete";

            form.appendChild(inp);
            form.appendChild(btn);
            right.appendChild(form);

            li.appendChild(left);
            li.appendChild(right);
            listEl.appendChild(li);
        }
    }

    async function loadDomains() {
        setLoading(true);
        msgEl.hidden = msgEl.textContent.trim().length === 0;

        try {
            const res = await fetch("/api/domains", { headers: { "Accept": "application/json" } });
            if (!res.ok) throw new Error("HTTP " + res.status);
            const data = await res.json();
            allDomains = Array.isArray(data) ? data : [];
            applyFilterAndRender();
        } catch (e) {
            showMessage("Failed to load domains (check /api/domains)");
            render([]);
        } finally {
            setLoading(false);
        }
    }

    // events
    clearBtn?.addEventListener("click", () => {
        domainsInput.value = "";
        domainsInput.focus();
    });

    searchInput?.addEventListener("input", () => {
        currentQuery = searchInput.value;
        applyFilterAndRender();
    });

    clearSearchBtn?.addEventListener("click", () => {
        searchInput.value = "";
        currentQuery = "";
        applyFilterAndRender();
        searchInput.focus();
    });

    // init
    const params = new URLSearchParams(window.location.search);
    showMessage(params.get("msg"));
    loadDomains();
})();
