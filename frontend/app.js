/* ============================================================
   DOUXOS — APPLICATION LOGIC
   Canvas network visualization + API integration
   ============================================================ */

(() => {
    'use strict';

    // ---- CONFIG ----
    const API_BASE = '';
    const POLL_INTERVAL = 1200;
    const COLORS = {
        green: '#00ff41',
        greenDim: '#00cc33',
        greenMuted: '#0a3d1a',
        cyan: '#00d4ff',
        cyanDim: '#0088aa',
        red: '#ff0040',
        white: '#e0e0e0',
        dim: '#555555',
        dimmer: '#333333',
        border: '#1a1a1a',
        black: '#000000',
    };

    // ---- STATE ----
    const state = {
        jobId: null,
        status: 'IDLE',
        totalChunks: 0,
        completedChunks: 0,
        progress: 0,
        polling: false,
        selectedFile: null,
        selectedScript: null,
        nodes: [],
        particles: [],
        chunks: [],
        startTime: Date.now(),
    };

    // ---- DOM REFS ----
    const $ = (s) => document.querySelector(s);
    const $$ = (s) => document.querySelectorAll(s);

    const els = {
        cursor: $('#cursor'),
        cursorTrail: $('#cursor-trail'),
        headerStatus: $('#headerStatus'),
        headerTime: $('#headerTime'),
        uploadZone: $('#uploadZone'),
        fileInput: $('#fileInput'),
        scriptInput: $('#scriptInput'),
        scriptBtn: $('#scriptBtn'),
        scriptName: $('#scriptName'),
        deployBtn: $('#deployBtn'),
        statJobId: $('#statJobId'),
        statStatus: $('#statStatus'),
        statCompleted: $('#statCompleted'),
        statTotal: $('#statTotal'),
        statProgress: $('#statProgress'),
        progressFill: $('#progressFill'),
        progressGlow: $('#progressGlow'),
        canvas: $('#networkCanvas'),
        canvasContainer: $('#canvasContainer'),
        logContainer: $('#logContainer'),
        logClearBtn: $('#logClearBtn'),
        downloadPanel: $('#download-panel'),
        downloadBtn: $('#downloadBtn'),
        nodeCount: $('#nodeCount'),
        footerConn: $('#footerConn'),
    };

    // ---- CUSTOM CURSOR ----
    let mouseX = 0, mouseY = 0;
    let trailX = 0, trailY = 0;

    document.addEventListener('mousemove', (e) => {
        mouseX = e.clientX;
        mouseY = e.clientY;
        els.cursor.style.left = mouseX + 'px';
        els.cursor.style.top = mouseY + 'px';
    });

    function animateCursor() {
        trailX += (mouseX - trailX) * 0.12;
        trailY += (mouseY - trailY) * 0.12;
        els.cursorTrail.style.left = trailX + 'px';
        els.cursorTrail.style.top = trailY + 'px';
        requestAnimationFrame(animateCursor);
    }
    animateCursor();

    // Hover state
    document.addEventListener('mouseover', (e) => {
        const hoverable = e.target.closest('button, .upload-zone, a, .script-btn');
        if (hoverable) els.cursor.classList.add('hovering');
    });
    document.addEventListener('mouseout', (e) => {
        const hoverable = e.target.closest('button, .upload-zone, a, .script-btn');
        if (hoverable) els.cursor.classList.remove('hovering');
    });

    // ---- CLOCK ----
    function updateClock() {
        const now = new Date();
        els.headerTime.textContent =
            String(now.getHours()).padStart(2, '0') + ':' +
            String(now.getMinutes()).padStart(2, '0') + ':' +
            String(now.getSeconds()).padStart(2, '0');
    }
    setInterval(updateClock, 1000);
    updateClock();

    // ---- UPLOAD ZONE ----
    els.uploadZone.addEventListener('click', () => els.fileInput.click());

    els.uploadZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        els.uploadZone.classList.add('dragover');
    });

    els.uploadZone.addEventListener('dragleave', () => {
        els.uploadZone.classList.remove('dragover');
    });

    els.uploadZone.addEventListener('drop', (e) => {
        e.preventDefault();
        els.uploadZone.classList.remove('dragover');
        if (e.dataTransfer.files.length) {
            state.selectedFile = e.dataTransfer.files[0];
            showFileSelected();
        }
    });

    els.fileInput.addEventListener('change', () => {
        if (els.fileInput.files.length) {
            state.selectedFile = els.fileInput.files[0];
            showFileSelected();
        }
    });

    function showFileSelected() {
        els.uploadZone.classList.add('has-file');
        const existingName = els.uploadZone.querySelector('.upload-filename');
        if (existingName) existingName.remove();

        const nameEl = document.createElement('div');
        nameEl.className = 'upload-filename';
        nameEl.textContent = state.selectedFile.name;
        els.uploadZone.appendChild(nameEl);

        const textEl = els.uploadZone.querySelector('.upload-text');
        if (textEl) textEl.textContent = 'FILE LOADED';

        checkDeployReady();
        addLog('File selected: ' + state.selectedFile.name, 'info');
    }

    // Script select
    els.scriptBtn.addEventListener('click', () => els.scriptInput.click());
    els.scriptInput.addEventListener('change', () => {
        if (els.scriptInput.files.length) {
            state.selectedScript = els.scriptInput.files[0];
            els.scriptName.textContent = state.selectedScript.name;
            els.scriptName.style.color = COLORS.cyan;
            addLog('Script selected: ' + state.selectedScript.name, 'info');
        }
    });

    function checkDeployReady() {
        els.deployBtn.disabled = !state.selectedFile;
    }

    // ---- DEPLOY ----
    els.deployBtn.addEventListener('click', async () => {
        if (!state.selectedFile) return;

        els.deployBtn.disabled = true;
        addLog('Deploying job...', 'system');
        updateHeaderStatus('DEPLOYING');

        const form = new FormData();
        form.append('file', state.selectedFile);
        if (state.selectedScript) {
            form.append('script', state.selectedScript);
        }

        try {
            const res = await fetch(API_BASE + '/api/upload', { method: 'POST', body: form });
            const data = await res.json();
            state.jobId = data.job_id;

            els.statJobId.textContent = state.jobId;
            els.statJobId.classList.add('active');

            addLog('Job created: ID ' + state.jobId, 'success');
            updateHeaderStatus('PROCESSING');

            // Start polling
            startPolling();

            // Generate visualization nodes
            generateNodes();

        } catch (err) {
            addLog('Deploy failed: ' + err.message, 'error');
            updateHeaderStatus('ERROR');
            els.deployBtn.disabled = false;
        }
    });

    // ---- POLLING ----
    function startPolling() {
        if (state.polling) return;
        state.polling = true;
        els.footerConn.textContent = 'POLLING';
        els.footerConn.classList.add('connected');
        pollStatus();
    }

    async function pollStatus() {
        if (!state.polling || !state.jobId) return;

        try {
            const res = await fetch(API_BASE + '/api/status/' + state.jobId);
            const data = await res.json();

            const prevCompleted = state.completedChunks;
            state.status = data.status;
            state.totalChunks = data.total_chunks;
            state.completedChunks = data.completed_chunks;
            state.progress = data.progress || 0;

            // Update UI
            updateStats();

            // Log new completions
            if (state.completedChunks > prevCompleted) {
                const diff = state.completedChunks - prevCompleted;
                for (let i = 0; i < diff; i++) {
                    addLog(`Chunk ${prevCompleted + i + 1}/${state.totalChunks} processed`, 'success');
                    // Fire particle
                    fireChunkParticle();
                }
            }

            if (state.status === 'COMPLETED') {
                state.polling = false;
                updateHeaderStatus('COMPLETED');
                addLog('All chunks processed. Job complete.', 'success');
                addLog('Output ready for download.', 'info');
                els.downloadPanel.classList.remove('hidden');
                completeAllNodes();
                return;
            }

            if (state.status === 'FAILED') {
                state.polling = false;
                updateHeaderStatus('FAILED');
                addLog('Job failed.', 'error');
                return;
            }

        } catch (err) {
            addLog('Poll error: ' + err.message, 'error');
        }

        setTimeout(pollStatus, POLL_INTERVAL);
    }

    function updateStats() {
        els.statStatus.textContent = state.status;
        els.statStatus.className = 'stat-value';
        if (state.status === 'PROCESSING') els.statStatus.classList.add('active');
        if (state.status === 'COMPLETED') els.statStatus.style.color = COLORS.cyan;

        els.statCompleted.textContent = state.completedChunks;
        els.statTotal.textContent = state.totalChunks;

        const pct = Math.round(state.progress);
        els.statProgress.textContent = pct + '%';
        els.progressFill.style.width = pct + '%';

        if (pct > 0 && pct < 100) {
            els.progressFill.parentElement.classList.add('active');
        }
        if (pct >= 100) {
            els.progressFill.style.background = COLORS.cyan;
        }
    }

    function updateHeaderStatus(text) {
        els.headerStatus.textContent = text;
        els.headerStatus.style.color =
            text === 'ERROR' || text === 'FAILED' ? COLORS.red :
            text === 'COMPLETED' ? COLORS.cyan :
            COLORS.green;
    }

    // ---- DOWNLOAD ----
    els.downloadBtn.addEventListener('click', () => {
        if (state.jobId) {
            window.open(API_BASE + '/api/download/' + state.jobId, '_blank');
        }
    });

    // ---- ACTIVITY LOG ----
    function addLog(msg, type = 'system') {
        const entry = document.createElement('div');
        entry.className = 'log-entry log-' + type;

        const now = new Date();
        const time = String(now.getHours()).padStart(2, '0') + ':' +
                     String(now.getMinutes()).padStart(2, '0') + ':' +
                     String(now.getSeconds()).padStart(2, '0');

        entry.innerHTML = `<span class="log-time">${time}</span><span class="log-msg">${msg}</span>`;
        els.logContainer.appendChild(entry);
        els.logContainer.scrollTop = els.logContainer.scrollHeight;
    }

    els.logClearBtn.addEventListener('click', () => {
        els.logContainer.innerHTML = '';
        addLog('Log cleared.', 'system');
    });

    // ============================================================
    // CANVAS NETWORK VISUALIZATION
    // ============================================================
    const ctx = els.canvas.getContext('2d');
    let canvasW, canvasH;
    let animFrame;
    let centerX, centerY;

    function resizeCanvas() {
        const rect = els.canvasContainer.getBoundingClientRect();
        const dpr = window.devicePixelRatio || 1;
        canvasW = rect.width;
        canvasH = rect.height;
        els.canvas.width = canvasW * dpr;
        els.canvas.height = canvasH * dpr;
        els.canvas.style.width = canvasW + 'px';
        els.canvas.style.height = canvasH + 'px';
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        centerX = canvasW / 2;
        centerY = canvasH / 2;
    }

    window.addEventListener('resize', resizeCanvas);
    resizeCanvas();

    // ---- NODE OBJECTS ----
    function generateNodes() {
        state.nodes = [];
        const count = Math.max(state.totalChunks, 4);
        const actualCount = Math.min(count, 12);
        const radius = Math.min(canvasW, canvasH) * 0.34;

        for (let i = 0; i < actualCount; i++) {
            const angle = (Math.PI * 2 / actualCount) * i - Math.PI / 2;
            state.nodes.push({
                x: centerX + Math.cos(angle) * radius,
                y: centerY + Math.sin(angle) * radius,
                targetX: centerX + Math.cos(angle) * radius,
                targetY: centerY + Math.sin(angle) * radius,
                size: 18,
                label: 'N-' + String(i + 1).padStart(2, '0'),
                active: false,
                completed: false,
                pulse: 0,
                angle: angle,
                birthTime: Date.now() + i * 120,
                opacity: 0,
            });
        }

        els.nodeCount.textContent = actualCount + ' NODES';
        addLog(actualCount + ' compute nodes initialized.', 'info');
    }

    function fireChunkParticle() {
        // Pick a random active node
        const available = state.nodes.filter(n => !n.completed);
        if (available.length === 0) return;

        const node = available[Math.floor(Math.random() * available.length)];
        node.active = true;

        // Particle from center to node
        state.particles.push({
            x: centerX,
            y: centerY,
            tx: node.x,
            ty: node.y,
            progress: 0,
            speed: 0.015 + Math.random() * 0.01,
            color: COLORS.green,
            size: 3,
            type: 'outgoing',
        });

        // After delay, particle from node back to center
        setTimeout(() => {
            node.active = false;
            node.completed = true;
            node.pulse = 1;

            state.particles.push({
                x: node.x,
                y: node.y,
                tx: centerX,
                ty: centerY,
                progress: 0,
                speed: 0.02 + Math.random() * 0.01,
                color: COLORS.cyan,
                size: 3,
                type: 'returning',
            });
        }, 800);
    }

    function completeAllNodes() {
        state.nodes.forEach(n => {
            n.completed = true;
            n.active = false;
        });
    }

    // ---- AMBIENT PARTICLES ----
    const ambientParticles = [];

    function initAmbientParticles() {
        for (let i = 0; i < 40; i++) {
            ambientParticles.push({
                x: Math.random() * canvasW,
                y: Math.random() * canvasH,
                vx: (Math.random() - 0.5) * 0.3,
                vy: (Math.random() - 0.5) * 0.3,
                size: Math.random() * 1.5 + 0.5,
                opacity: Math.random() * 0.3 + 0.05,
            });
        }
    }
    initAmbientParticles();

    // ---- GRID BACKGROUND ----
    function drawGrid() {
        ctx.strokeStyle = 'rgba(26, 26, 26, 0.5)';
        ctx.lineWidth = 0.5;
        const step = 40;

        for (let x = step; x < canvasW; x += step) {
            ctx.beginPath();
            ctx.moveTo(x, 0);
            ctx.lineTo(x, canvasH);
            ctx.stroke();
        }
        for (let y = step; y < canvasH; y += step) {
            ctx.beginPath();
            ctx.moveTo(0, y);
            ctx.lineTo(canvasW, y);
            ctx.stroke();
        }
    }

    // ---- DRAW HEXAGON ----
    function drawHexagon(x, y, size, fill, stroke, lineWidth = 1) {
        ctx.beginPath();
        for (let i = 0; i < 6; i++) {
            const angle = (Math.PI / 3) * i - Math.PI / 6;
            const px = x + size * Math.cos(angle);
            const py = y + size * Math.sin(angle);
            if (i === 0) ctx.moveTo(px, py);
            else ctx.lineTo(px, py);
        }
        ctx.closePath();
        if (fill) {
            ctx.fillStyle = fill;
            ctx.fill();
        }
        if (stroke) {
            ctx.strokeStyle = stroke;
            ctx.lineWidth = lineWidth;
            ctx.stroke();
        }
    }

    // ---- DRAW FRAME ----
    function drawFrame() {
        const now = Date.now();
        ctx.clearRect(0, 0, canvasW, canvasH);

        // Grid
        drawGrid();

        // Ambient particles
        ambientParticles.forEach(p => {
            p.x += p.vx;
            p.y += p.vy;
            if (p.x < 0) p.x = canvasW;
            if (p.x > canvasW) p.x = 0;
            if (p.y < 0) p.y = canvasH;
            if (p.y > canvasH) p.y = 0;

            ctx.fillStyle = `rgba(0, 255, 65, ${p.opacity})`;
            ctx.fillRect(p.x, p.y, p.size, p.size);
        });

        // Connection lines to nodes
        state.nodes.forEach(node => {
            // Fade-in
            const age = now - node.birthTime;
            node.opacity = Math.min(1, Math.max(0, age / 600));
            if (node.opacity <= 0) return;

            ctx.save();
            ctx.globalAlpha = node.opacity;

            // Connection line
            const lineColor = node.completed ? COLORS.cyan :
                              node.active ? COLORS.green : COLORS.dimmer;
            ctx.strokeStyle = lineColor;
            ctx.lineWidth = node.active ? 1.5 : 0.5;
            ctx.setLineDash(node.active ? [] : [4, 4]);
            ctx.beginPath();
            ctx.moveTo(centerX, centerY);
            ctx.lineTo(node.x, node.y);
            ctx.stroke();
            ctx.setLineDash([]);

            // Node hexagon
            const nodeFill = node.completed ? 'rgba(0, 212, 255, 0.08)' :
                             node.active ? 'rgba(0, 255, 65, 0.08)' :
                             'rgba(10, 10, 10, 0.8)';
            const nodeStroke = node.completed ? COLORS.cyan :
                               node.active ? COLORS.green : COLORS.dimmer;

            drawHexagon(node.x, node.y, node.size, nodeFill, nodeStroke, node.active ? 2 : 1);

            // Pulse ring
            if (node.pulse > 0) {
                node.pulse -= 0.015;
                const pulseSize = node.size + (1 - node.pulse) * 30;
                ctx.globalAlpha = node.pulse * node.opacity;
                drawHexagon(node.x, node.y, pulseSize, null, COLORS.cyan, 1);
            }

            // Label
            ctx.globalAlpha = node.opacity * 0.7;
            ctx.fillStyle = nodeStroke;
            ctx.font = '500 9px "JetBrains Mono"';
            ctx.textAlign = 'center';
            ctx.fillText(node.label, node.x, node.y + node.size + 16);

            ctx.restore();
        });

        // Center node (ORACLE)
        const oracleSize = 32;
        const oraclePulse = Math.sin(now * 0.002) * 0.15 + 0.85;

        // Outer ring
        ctx.save();
        ctx.globalAlpha = 0.15 * oraclePulse;
        drawHexagon(centerX, centerY, oracleSize + 12, null, COLORS.green, 1);
        ctx.restore();

        // Inner
        drawHexagon(centerX, centerY, oracleSize, 'rgba(0, 255, 65, 0.04)', COLORS.green, 1.5);

        // Oracle label
        ctx.fillStyle = COLORS.green;
        ctx.font = '700 11px "JetBrains Mono"';
        ctx.textAlign = 'center';
        ctx.fillText('ORACLE', centerX, centerY + 4);

        // Crosshair
        ctx.strokeStyle = 'rgba(0, 255, 65, 0.15)';
        ctx.lineWidth = 0.5;
        ctx.beginPath();
        ctx.moveTo(centerX - oracleSize - 20, centerY);
        ctx.lineTo(centerX - oracleSize - 5, centerY);
        ctx.moveTo(centerX + oracleSize + 5, centerY);
        ctx.lineTo(centerX + oracleSize + 20, centerY);
        ctx.moveTo(centerX, centerY - oracleSize - 20);
        ctx.lineTo(centerX, centerY - oracleSize - 5);
        ctx.moveTo(centerX, centerY + oracleSize + 5);
        ctx.lineTo(centerX, centerY + oracleSize + 20);
        ctx.stroke();

        // Data particles
        state.particles = state.particles.filter(p => {
            p.progress += p.speed;
            if (p.progress >= 1) return false;

            const eased = easeInOutCubic(p.progress);
            const px = p.x + (p.tx - p.x) * eased;
            const py = p.y + (p.ty - p.y) * eased;

            // Trail
            const trailLen = 5;
            for (let t = 0; t < trailLen; t++) {
                const tp = Math.max(0, p.progress - t * 0.02);
                const te = easeInOutCubic(tp);
                const tx = p.x + (p.tx - p.x) * te;
                const ty = p.y + (p.ty - p.y) * te;
                ctx.fillStyle = p.color;
                ctx.globalAlpha = (1 - t / trailLen) * 0.6;
                ctx.fillRect(tx - 1, ty - 1, 2, 2);
            }

            // Main dot
            ctx.globalAlpha = 1;
            ctx.fillStyle = p.color;
            ctx.fillRect(px - p.size / 2, py - p.size / 2, p.size, p.size);

            // Glow
            ctx.globalAlpha = 0.3;
            ctx.fillRect(px - p.size, py - p.size, p.size * 2, p.size * 2);

            ctx.globalAlpha = 1;
            return true;
        });

        // Corner decorations
        drawCorner(8, 8, 20, 'tl');
        drawCorner(canvasW - 8, 8, 20, 'tr');
        drawCorner(8, canvasH - 8, 20, 'bl');
        drawCorner(canvasW - 8, canvasH - 8, 20, 'br');

        // Status text in bottom
        ctx.fillStyle = COLORS.dimmer;
        ctx.font = '400 9px "JetBrains Mono"';
        ctx.textAlign = 'left';
        ctx.fillText('SYS:NETWORK_TOPOLOGY', 24, canvasH - 16);

        ctx.textAlign = 'right';
        const elapsed = Math.floor((now - state.startTime) / 1000);
        const mins = String(Math.floor(elapsed / 60)).padStart(2, '0');
        const secs = String(elapsed % 60).padStart(2, '0');
        ctx.fillText('T+' + mins + ':' + secs, canvasW - 24, canvasH - 16);

        animFrame = requestAnimationFrame(drawFrame);
    }

    function drawCorner(x, y, size, pos) {
        ctx.strokeStyle = COLORS.dimmer;
        ctx.lineWidth = 1;
        ctx.beginPath();
        if (pos === 'tl') {
            ctx.moveTo(x, y + size);
            ctx.lineTo(x, y);
            ctx.lineTo(x + size, y);
        } else if (pos === 'tr') {
            ctx.moveTo(x - size, y);
            ctx.lineTo(x, y);
            ctx.lineTo(x, y + size);
        } else if (pos === 'bl') {
            ctx.moveTo(x, y - size);
            ctx.lineTo(x, y);
            ctx.lineTo(x + size, y);
        } else {
            ctx.moveTo(x - size, y);
            ctx.lineTo(x, y);
            ctx.lineTo(x, y - size);
        }
        ctx.stroke();
    }

    function easeInOutCubic(t) {
        return t < 0.5
            ? 4 * t * t * t
            : 1 - Math.pow(-2 * t + 2, 3) / 2;
    }

    // Start render loop
    drawFrame();

    // ---- SMOOTH SCROLL HERO -> DASHBOARD ----
    const hero = $('#hero');
    hero.addEventListener('click', () => {
        $('#dashboard').scrollIntoView({ behavior: 'smooth' });
    });

    // ---- ENTRY ANIMATIONS ----
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, { threshold: 0.1 });

    $$('.panel').forEach((panel, i) => {
        panel.style.opacity = '0';
        panel.style.transform = 'translateY(20px)';
        panel.style.transition = `opacity 0.6s cubic-bezier(0.16, 1, 0.3, 1) ${i * 0.1}s, transform 0.6s cubic-bezier(0.16, 1, 0.3, 1) ${i * 0.1}s`;
        observer.observe(panel);
    });

})();
