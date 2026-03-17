(() => {
    const msgEl = document.getElementById("msg");
    const countEl = document.getElementById("count");
    const listEl = document.getElementById("list");
    const loadingEl = document.getElementById("loading");
    const emptyEl = document.getElementById("empty");
    const addBtn = document.getElementById("addBtn");
    const clearBtn = document.getElementById("clearBtn");
    const domainsInput = document.getElementById("domainsInput");
    const searchInput = document.getElementById("searchInput");
    const clearSearchBtn = document.getElementById("clearSearchBtn");
    const copyArea = document.getElementById("domainsPlainTxt");

    let allDomains = [];
    let filteredDomains = [];
    let currentQuery = "";

    function showMessage(text, isError = false) {
        if (!text) {
            msgEl.hidden = true;
            return;
        }

        msgEl.textContent = text;
        if (isError === true)
            msgEl.classList.add("err")
        else
            msgEl.classList.remove("err");

        msgEl.hidden = false;
    }

    async function makeRequest(url, method, data = null) {
        const options = {
            method: method,
            headers: {
                "Accept": "application/json"
            }
        };

        if (data != null) {
            options.body = JSON.stringify(data);
            options.headers["Content-Type"] = "application/json";
        }
        else data = undefined;

        try {
            var response = await fetch(url, options);
            var resultData = null;
            try {
                const contentType = response.headers.get("Content-Type");
                if (responce.status != 204 && contentType?.includes("application/json")) {
                    resultData = await response.json();
                    isJson = resultData != null;
                }
            } catch (e) {
                console.error(e);
                showMessage(e.message, true);
            }

            var message = resultData?.message;
            if (message != null) {
                showMessage(message, !response.ok);
            } else if (!response.ok) {
                if (response.status === 401 || response.status === 403) {
                    message = "Unauthorized";
                } else if (response.status
                    === 404) {
                    message = "Not Found";
                } else if (response.status >= 500) {
                    message = "Server error";
                } else if (response.status === 400 || response.status >= 404) {
                    message = "Invalid request";
                }
                showMessage(message, true);
            }
        } catch (e) {
            console.error(e);
            showMessage(e.message, true);
        }
        return { status: response.status, data: resultData };
    }

    function filterDomains(domains, q) {
        const query = (q || "").trim().toLowerCase();
        if (!query) {
            filteredDomains = domains;
            return domains;
        }
        filteredDomains = domains.filter(d => d.toLowerCase().includes(query));
        return filteredDomains;
    }

    function render(domains) {
        emptyEl.hidden = domains.length > 0;

        const existingNodes = new Map();
        Array.from(listEl.children).forEach(node => {
            const domain = node.dataset.domain;
            if (domain) existingNodes.set(domain, node);
        });

        const activeDomainSet = new Set(domains);
        for (const [domain, node] of existingNodes) {
            if (!activeDomainSet.has(domain)) {
                node.remove();
                existingNodes.delete(domain);
            }
        }

        domains.forEach(d => {
            let node = existingNodes.get(d);
            if (!node) {
                node = createDomainItem(d);
            }
            listEl.prepend(node);
        });
    }

    function createDomainItem(domain) {
        const li = document.createElement("li");
        li.className = "item";
        li.dataset.domain = domain;

        const left = document.createElement("div");
        left.className = "domain";
        left.textContent = domain;

        const right = document.createElement("div");
        right.className = "actions";

        const btn = document.createElement("button");
        btn.className = "btn danger";
        btn.dataset.domain = domain;
        btn.textContent = "Delete";
        btn.addEventListener("click", deleteHandler);

        right.appendChild(btn);
        li.appendChild(left);
        li.appendChild(right);

        return li;
    }

    function applyFilterAndRender() {
        const filtered = filterDomains(allDomains, currentQuery);
        render(filtered);
    }

    async function loadDomains() {
        var response = await makeRequest("/api/list", "GET");
        let data = response.data?.domains;
        allDomains = Array.isArray(data) ? data : [];
        countEl.textContent = String(allDomains.length);
        applyFilterAndRender();
        copyArea.value = "";
    }

    async function addHandler(event) {
        domains = domainsInput.value.split(/\s+/);
        var response = await makeRequest("/api/add", "POST", { domains: domains });
        if (response.status === 200) {
            await loadDomains();
            clearInputHandler();
        }
    }

    function clearInputHandler(event) {
        domainsInput.value = "";
        domainsInput.focus();
    }

    function copyAllHandler(event) {
        const mode = event.target.dataset.copyall;
        let lines = [];

        switch (mode) {
            case 'dnsmasq':
                lines = filteredDomains.map(d => `server=/${d}/10.99.99.1`);
                break;
            case 'adguard':
                lines = filteredDomains.map(d => `[/${d}/]10.99.99.1:54`);
                break;
            case 'noip':
            default:
                lines = filteredDomains;
                break;
        }

        copyArea.value = lines.join('\n');
    }

    async function serviceActionHandler(event) {
        const action = event.target.dataset.action;
        await makeRequest("/api/action", "POST", { action: action });
    }

    async function deleteHandler(event) {
        if (!confirm(`Delete ${event.target.dataset.domain}?`)) return;
        var response = await makeRequest("/api/delete", "POST", { domain: event.target.dataset.domain });
        if (response.status === 200) {
            await loadDomains();
            clearInputHandler();
        }
    }

    // events
    addBtn.addEventListener("click", addHandler);
    clearBtn.addEventListener("click", clearInputHandler);
    document.querySelectorAll("button[data-copyall]").forEach(b => b.addEventListener("click", copyAllHandler));
    document.querySelectorAll("button[data-action]").forEach(b => b.addEventListener("click", serviceActionHandler));

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
    loadDomains();
})();
