import './style.css';
import htmlTemplate from './template.html?raw';

import { SelectFolder, StartExecution, ShowResults, CancelExecution, CheckIfResultsExist, GetResults, RevealInExplorer } from '../wailsjs/go/processing/FrontendApp';
import { FrontEnd_DuplicateGroup } from './models.js';

document.querySelector('#app').innerHTML = htmlTemplate;

// --- Elements for Status Update ---
const progressTitle = document.getElementById("progress-title");
const progressBar = document.getElementById("progress-bar");
const detailedStatus = document.getElementById("detailed-status");
const showResultsButton = document.getElementById('showResultsButton');

const startButton = document.getElementById('startButton');
const stopButton = document.getElementById('stopButton');

const startText = document.getElementById('startText');
const startButtonSpinner = document.getElementById('startButtonSpinner');

// --- Duplicate Results State ---
const PAGE_SIZE = 3;
let allGroups = [];
let currentPage = 1;

const resultsSection = document.getElementById('results-section');
const resultsList = document.getElementById('results-list');
const resultsCountLabel = document.getElementById('results-count-label');
const prevPageTop = document.getElementById('prev-page-top');
const nextPageTop = document.getElementById('next-page-top');
const pageIndicatorTop = document.getElementById('page-indicator-top');
const prevPageBottom = document.getElementById('prev-page-bottom');
const nextPageBottom = document.getElementById('next-page-bottom');
const pageIndicatorBottom = document.getElementById('page-indicator-bottom');
const resultsControlsTop = document.getElementById('results-controls-top');
const resultsControlsBottom = document.getElementById('results-controls-bottom');

// --- Directory Selection Handler ---
/**
 * Opens a folder selection dialog and sets the input field's value.
 * @param {string} inputId The ID of the input field to update.
 */
window.selectAndSetDir = function (inputId) {
    SelectFolder()
        .then((path) => {
            if (path) {
                document.getElementById(inputId).value = path;
            }
        })
        .catch((err) => {
            console.error("Directory selection error:", err);
        });
};

// --- Collapsible Section Handler ---
window.toggleAdvanced = function () {
    const section = document.getElementById('advancedSection');
    const content = document.getElementById('advancedContent');

    if (!section || !content) return;

    const isCollapsed = section.getAttribute('data-collapsed') === 'true';

    if (isCollapsed) {
        section.setAttribute('data-collapsed', 'false');

        content.style.paddingTop = '15px';
        content.style.paddingBottom = '15px';

        const contentHeight = content.scrollHeight;

        content.style.maxHeight = (contentHeight + 50) + 'px';

    } else {
        content.style.maxHeight = content.scrollHeight + 'px';

        void content.offsetWidth;

        section.setAttribute('data-collapsed', 'true');
        content.style.maxHeight = '0';
        content.style.paddingTop = '0';
        content.style.paddingBottom = '0';
    }
};

// --- Execution Start Handler ---
window.startProcess = function () {
    // 1. Gather data
    const params = {
        sourceDir: document.getElementById('sourceDir').value,
        targetDir: document.getElementById('targetDir').value,
        useCache: document.getElementById('keepMemory').checked,
        cacheDir: document.getElementById('cacheDir').value,
        resultsDir: document.getElementById('resultsDir').value,
        paranoidMode: document.getElementById('paranoidMode').checked,
        cpus: parseInt(document.getElementById('cpus').value) || 0,
        bufSize: parseInt(document.getElementById('bufSize').value) || 0,
        debugMode: document.getElementById('debugMode').checked,
        dualFolderModeEnabled: false,
    };

    // Clear old status/reset bar
    progressTitle.innerText = "Starting up...";
    detailedStatus.innerHTML = "";
    progressBar.style.width = '0%';
    // Reset color to the primary accent color (bright green)
    progressBar.style.backgroundColor = 'var(--color-accent)';

    // Hide previous results
    resultsSection.style.display = 'none';
    allGroups = [];
    resultsList.innerHTML = '';


    // UI State: Running
    startButton.disabled = true;
    stopButton.disabled = false;
    showResultsButton.disabled = true;

    toggleStartSpinner(true);

    // 2. Call the Go backend function
    StartExecution(params)
        .then((result) => {
            detailedStatus.innerHTML += `<div>${result}</div>`;
            startButton.disabled = false; // Enable Start again
            stopButton.disabled = true;   // Disable Stop
            toggleStartSpinner(false);
        })
        .catch((err) => {
            progressTitle.innerText = `FATAL BINDING ERROR`;
            detailedStatus.innerHTML = `<div style="color: #E57373;">${err}</div>`;
            // Ensure buttons reset on binding error
            startButton.disabled = false;
            stopButton.disabled = true;
            toggleStartSpinner(false);
        });
};

// --- Execution Cancellation Handler ---
window.cancelProcess = function () {
    progressTitle.innerText = "Cancellation Requested...";
    detailedStatus.innerHTML += '<div style="color: #FFB74D; font-weight: bold;">[INFO] Sending cancellation signal to backend...</div>';

    // Disable the stop button immediately to prevent multiple presses
    stopButton.disabled = true;

    CancelExecution()
        .then(() => {
            // Backend received signal. The actual termination will be reflected by the status listener.
            progressTitle.innerText = "Process Stopped.";
            startButton.disabled = false; // Allow restart
            toggleStartSpinner(false);
        })
        .catch((err) => {
            // Should generally not happen if binding is correct, but good to handle.
            progressTitle.innerText = `CANCELLATION ERROR`;
            detailedStatus.innerHTML += `<div style="color: #E57373;">[ERROR] Failed to send cancellation: ${err}</div>`;
            startButton.disabled = false;
            toggleStartSpinner(false);
        });
};

window.showResults = function () {
    ShowResults()
        .catch((err) => {
            detailedStatus.innerHTML += `<div style="color: #E57373; font-weight: bold;">[ERROR] Failed to open results file: ${err}</div>`;
        });
};

const MAX_LOG_ROWS = 100; // Define your maximum row limit (e.g., 100 rows) [less things on screen less lag]

// --- Status Listener Setup ---
function setupStatusListeners() {
    const showResultsButton = document.getElementById('showResultsButton'); // Get the element again
    // 1. Progress/Title Update Event
    runtime.EventsOn("progressUpdate", (data) => {
        if (progressTitle.innerText != data.title) {
            progressTitle.innerText = data.title;
        }
        if (data.percent !== undefined) {
            // Ensure the value is treated as a number and cap it
            const rawPercent = parseFloat(data.percent);
            const cappedPercent = Math.min(100, Math.max(0, rawPercent));


            const displayPercent = cappedPercent.toFixed(2);

            progressBar.style.width = `${cappedPercent}%`;
            progressBar.innerText = cappedPercent > 5 ? `${displayPercent}%` : '';
            if (cappedPercent >= 100) {
                progressTitle.innerText = "Process Complete.";
                toggleStartSpinner(false);
                refreshResultsButtonState();

            } else {
                showResultsButton.disabled = true;
            }
        }
    });

    // 2. Detailed Log Event
    runtime.EventsOn("detailedLog", (message) => {

        const tolerance = 20;
        const isScrolledToBottom = (detailedStatus.scrollHeight - detailedStatus.clientHeight) <= (detailedStatus.scrollTop + tolerance);

        const logEntry = document.createElement('div');
        logEntry.innerHTML = `[${new Date().toLocaleTimeString()}] ${message}`;

        detailedStatus.appendChild(logEntry);

        while (detailedStatus.children.length > MAX_LOG_ROWS) {
            detailedStatus.removeChild(detailedStatus.firstChild);
        }

        // 4. SCROLL: Only scroll to bottom if the user was already at the bottom
        if (isScrolledToBottom) {
            detailedStatus.scrollTop = detailedStatus.scrollHeight;
        }

    });

    // 3. Error Event
    runtime.EventsOn("errorUpdate", (message) => {
        progressTitle.innerText = "Error: Process Failed";
        detailedStatus.innerHTML += `<div style="color: #E57373; font-weight: bold;">[ERROR] ${message}</div>`;
        progressBar.style.width = '100%';
        progressBar.style.backgroundColor = '#ff6b6b';

        showResultsButton.disabled = true;

        toggleStartSpinner(false);
    });

    runtime.EventsOn("executionFinished", (filePath) => {
    progressTitle.innerText = "Process Complete.";
    toggleStartSpinner(false);
    startButton.disabled = false;
    stopButton.disabled = true;

    // Fetch and display duplicate groups
    GetResults()
        .then(groups => renderResults(groups))
        .catch(err => console.error('GetResults error:', err));
});
}
// Run setup after DOM load
setupStatusListeners();
refreshResultsButtonState();

// --- Results: public page navigation (called from template onclick) ---
window.currentPage = currentPage; // expose for onclick expressions
window.goToPage = function (page) {
    renderPage(page);
};

// --- Results: reveal a specific file in the OS file manager ---
window.revealInExplorer = function (path) {
    RevealInExplorer(path)
        .catch(err => console.error('RevealInExplorer error:', err));
};

/**
 * Builds and returns a single duplicate-group card DOM node.
 * @param {FrontEnd_DuplicateGroup} group
 */
function createResultCard(group) {
    const card = document.createElement('div');
    card.className = 'result-card';

    const dupCount = group.duplicates ? group.duplicates.length : 0;

    // --- Header ---
    const header = document.createElement('div');
    header.className = 'result-card-header';

    const fileInfo = document.createElement('div');
    fileInfo.className = 'result-file-info';

    const nameSpan = document.createElement('span');
    nameSpan.className = 'result-filename';
    nameSpan.textContent = group.fileName;

    const pathSpan = document.createElement('span');
    pathSpan.className = 'result-filepath';
    pathSpan.textContent = group.filePath;
    pathSpan.title = group.filePath;

    fileInfo.appendChild(nameSpan);
    fileInfo.appendChild(pathSpan);

    const actions = document.createElement('div');
    actions.className = 'result-card-actions';

    const showBtn = document.createElement('button');
    showBtn.className = 'btn btn-show';
    showBtn.textContent = 'Show';
    showBtn.onclick = () => window.revealInExplorer(group.filePath);

    const dupLabel = `${dupCount} duplicate${dupCount !== 1 ? 's' : ''}`;
    const toggleBtn = document.createElement('button');
    toggleBtn.className = 'btn btn-toggle';
    toggleBtn.textContent = `\u25bc ${dupLabel}`;
    toggleBtn.onclick = () => {
        const isOpen = card.classList.toggle('is-open');
        toggleBtn.textContent = isOpen ? `\u25b2 hide` : `\u25bc ${dupLabel}`;
    };

    actions.appendChild(showBtn);
    actions.appendChild(toggleBtn);
    header.appendChild(fileInfo);
    header.appendChild(actions);

    // --- Body (collapsible duplicates list) ---
    const body = document.createElement('div');
    body.className = 'result-card-body';

    if (group.duplicates && group.duplicates.length > 0) {
        group.duplicates.forEach(dup => {
            const dupItem = document.createElement('div');
            dupItem.className = 'duplicate-item';

            const dupInfo = document.createElement('div');
            dupInfo.className = 'result-file-info';

            const dupName = document.createElement('span');
            dupName.className = 'result-filename';
            dupName.textContent = dup.fileName;

            const dupPath = document.createElement('span');
            dupPath.className = 'result-filepath';
            dupPath.textContent = dup.filePath;
            dupPath.title = dup.filePath;

            dupInfo.appendChild(dupName);
            dupInfo.appendChild(dupPath);

            const dupShowBtn = document.createElement('button');
            dupShowBtn.className = 'btn btn-show';
            dupShowBtn.textContent = 'Show';
            dupShowBtn.onclick = () => window.revealInExplorer(dup.filePath);

            dupItem.appendChild(dupInfo);
            dupItem.appendChild(dupShowBtn);
            body.appendChild(dupItem);
        });
    }

    card.appendChild(header);
    card.appendChild(body);
    return card;
}

/**
 * Renders one page of duplicate groups into the results list.
 * @param {number} page 1-based page number
 */
function renderPage(page) {
    const totalPages = Math.max(1, Math.ceil(allGroups.length / PAGE_SIZE));
    currentPage = Math.max(1, Math.min(page, totalPages));
    window.currentPage = currentPage;

    const start = (currentPage - 1) * PAGE_SIZE;
    const end = Math.min(start + PAGE_SIZE, allGroups.length);
    const pageGroups = allGroups.slice(start, end);

    resultsList.innerHTML = '';
    pageGroups.forEach(group => resultsList.appendChild(createResultCard(group)));

    const pageText = `Page ${currentPage} of ${totalPages}`;
    pageIndicatorTop.textContent = pageText;
    pageIndicatorBottom.textContent = pageText;

    prevPageTop.disabled = currentPage <= 1;
    prevPageBottom.disabled = currentPage <= 1;
    nextPageTop.disabled = currentPage >= totalPages;
    nextPageBottom.disabled = currentPage >= totalPages;

    const showPager = totalPages > 1;
    resultsControlsTop.style.display = showPager ? 'flex' : 'none';
    resultsControlsBottom.style.display = showPager ? 'flex' : 'none';

    resultsSection.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

/**
 * Populates and shows the results panel from a raw array of backend FileHash objects.
 * Maps each entry to the frontend DuplicateGroup model before storing.
 * @param {Array} rawGroups - backend models.FileHash[] from GetResults()
 */
function renderResults(rawGroups) {
    allGroups = (rawGroups || []).map(FrontEnd_DuplicateGroup.fromFileHash);
    currentPage = 1;
    window.currentPage = 1;

    if (allGroups.length === 0) {
        resultsSection.style.display = 'none';
        return;
    }

    resultsCountLabel.textContent =
        `${allGroups.length} duplicate group${allGroups.length !== 1 ? 's' : ''} found`;
    resultsSection.style.display = 'block';
    renderPage(1);
}

// --- Spinner State Handler ---
/**
 * Toggles the visibility of the start button text and spinner.
 * @param {boolean} isRunning - True if the process is starting/running.
 */
function toggleStartSpinner(isLoading) {
    if (isLoading) {
        startButton.classList.add('is-loading');
 
        startButtonSpinner.textContent = ''; 
    } else {
        startButton.classList.remove('is-loading');
        startButtonSpinner.textContent = '';
    }
}

// Function to check if the results button should be enabled.
async function refreshResultsButtonState() {
    try {
        // Assuming CheckIfResultsExist is exposed via Wails
        const exists = await CheckIfResultsExist();
        console.log("exists ", exists)
        showResultsButton.disabled = !exists;
    } catch (err) {
        showResultsButton.disabled = true;
    }
}