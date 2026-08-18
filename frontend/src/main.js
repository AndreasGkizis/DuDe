import './style.css';
import htmlTemplate from './template.html?raw';

import { SelectFolder, StartExecution, ShowResults, CancelExecution, CheckIfResultsExist, GetResults, RevealInExplorer, FullReset } from '../wailsjs/go/processing/FrontendApp';
import { FrontEnd_DuplicateGroup } from './models.js';

document.querySelector('#app').innerHTML = htmlTemplate;

// --- Elements for Status Update ---
const progressBar = document.getElementById("progress-bar");
const statusJob = document.getElementById("status-job");
const statusFiles = document.getElementById("status-files");
const statusDuplicates = document.getElementById("status-duplicates");
const statusError = document.getElementById("status-error");
const showResultsButton = document.getElementById('showResultsButton');
const clearResultsButton = document.getElementById('clearResultsButton');

const startButton = document.getElementById('startButton');
const stopButton = document.getElementById('stopButton');
const fullResetButton = document.getElementById('fullResetButton');
const modeInputs = Array.from(document.querySelectorAll('input[name="operationMode"]'));
const duplicateFolders = document.getElementById('duplicateFolders');
const coverageFolders = document.getElementById('coverageFolders');
const sourceDir = document.getElementById('sourceDir');
const targetDir = document.getElementById('targetDir');
const folderRelationshipWarning = document.getElementById('folderRelationshipWarning');
const coverageSummary = document.getElementById('coverage-summary');
const coverageOutcome = document.getElementById('coverage-outcome');
const coverageCovered = document.getElementById('coverage-covered');
const coverageMissing = document.getElementById('coverage-missing');
const statusMetricLabel = document.getElementById('status-metric-label');
const pageSizeLabel = document.getElementById('page-size-label');

const startText = document.getElementById('startText');
const startButtonSpinner = document.getElementById('startButtonSpinner');

// --- Duplicate Results State ---
const DEFAULT_PAGE_SIZE = 3;
const PAGE_SIZE_OPTIONS = [3, 5, 10, 25, 50];
let allGroups = [];
let currentPage = 1;
let pageSize = DEFAULT_PAGE_SIZE;
let activePhase = '';
let currentMode = 'duplicates';
let executionActive = false;

const resultsSection = document.getElementById('results-section');
const resultsList = document.getElementById('results-list');
const resultsCountLabel = document.getElementById('results-count-label');
const resultsPageSize = document.getElementById('resultsPageSize');
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
 * Accepts either a string input-element ID (legacy, for cacheDir/resultsDir)
 * or a button element (for dir-row rows — finds the sibling input).
 * @param {string|HTMLElement} target The ID of the input field or the button element.
 */
window.selectAndSetDir = function (target) {
    SelectFolder()
        .then((path) => {
            if (!path) return;
            if (typeof target === 'string') {
                document.getElementById(target).value = path;
                if (target === 'sourceDir' || target === 'targetDir') {
                    updateFolderRelationshipWarning();
                }
            } else {
                // target is the Select button inside a .dir-row; find the sibling input
                target.closest('.dir-row').querySelector('input').value = path;
            }
        })
        .catch((err) => {
            console.error("Directory selection error:", err);
        });
};

// --- Operation Mode Handlers ---

function resetFolderSelections() {
    const dirList = document.getElementById('dirList');
    dirList.innerHTML = `
        <div class="dir-row" data-index="0">
            <input class="input dir-input" type="text" readonly placeholder="Nothing selected!">
            <button class="btn btn-select" onclick="selectAndSetDir(this)">Select</button>
            <button class="btn btn-remove-dir" onclick="removeDir(this)" disabled title="Remove directory">−</button>
        </div>`;
    sourceDir.value = '';
    targetDir.value = '';
    updateFolderRelationshipWarning();
}

function normalizeFolderPath(path) {
    let normalized = path.trim().replace(/\\/g, '/');
    if (normalized !== '/' && !/^[a-z]:\/$/i.test(normalized)) {
        normalized = normalized.replace(/\/+$/, '');
    }
    return /^[a-z]:\//i.test(normalized) ? normalized.toLowerCase() : normalized;
}

function foldersOverlap(sourcePath, targetPath) {
    const source = normalizeFolderPath(sourcePath);
    const target = normalizeFolderPath(targetPath);
    if (!source || !target) return false;
    const sourcePrefix = source.endsWith('/') ? source : `${source}/`;
    const targetPrefix = target.endsWith('/') ? target : `${target}/`;
    return source === target || source.startsWith(targetPrefix) || target.startsWith(sourcePrefix);
}

function updateFolderRelationshipWarning() {
    folderRelationshipWarning.hidden = !foldersOverlap(sourceDir.value, targetDir.value);
}

function updateModePresentation() {
    const isCoverage = currentMode === 'coverage';
    duplicateFolders.hidden = isCoverage;
    coverageFolders.hidden = !isCoverage;
    coverageSummary.hidden = !isCoverage;
    statusMetricLabel.textContent = isCoverage ? 'Source Coverage' : 'Duplicates Found';
    pageSizeLabel.textContent = isCoverage ? 'Missing files per page' : 'Groups per page';
    resultsCountLabel.textContent = isCoverage ? 'Missing Files' : 'Results';
    startText.textContent = 'Start';
    startButton.title = isCoverage ? 'Coverage processing will be enabled with the backend implementation.' : '';
    startButton.disabled = executionActive || isCoverage;
    showResultsButton.disabled = true;
}

function setExecutionActive(isActive) {
    executionActive = isActive;
    modeInputs.forEach(input => { input.disabled = isActive; });
    startButton.disabled = isActive || currentMode === 'coverage';
}

window.changeOperationMode = function (mode) {
    if (executionActive || (mode !== 'duplicates' && mode !== 'coverage')) {
        modeInputs.forEach(input => { input.checked = input.value === currentMode; });
        return;
    }

    currentMode = mode;
    resetFolderSelections();
    clearResults();
    updateModePresentation();
};

// --- Dynamic Directory List Handlers ---

/**
 * Appends a new directory row to #dirList and updates remove-button visibility.
 */
window.addDir = function () {
    const list = document.getElementById('dirList');
    const index = list.children.length;
    const row = document.createElement('div');
    row.className = 'dir-row';
    row.dataset.index = index;
    row.innerHTML = `
        <input class="input dir-input" type="text" readonly placeholder="Nothing selected!">
        <button class="btn btn-select" onclick="selectAndSetDir(this)">Select</button>
        <button class="btn btn-remove-dir" onclick="removeDir(this)" title="Remove directory">−</button>
    `;
    list.appendChild(row);
    _updateRemoveButtons();
};

/**
 * Removes a directory row and updates remove-button visibility.
 * @param {HTMLElement} btn The remove button that was clicked.
 */
window.removeDir = function (btn) {
    const row = btn.closest('.dir-row');
    row.remove();
    _updateRemoveButtons();
};

/**
 * Steps a themed number input while preserving its native min/max behavior.
 * @param {string} inputId The number input to update.
 * @param {number} direction Positive to increment, negative to decrement.
 */
window.stepNumberInput = function (inputId, direction) {
    const input = document.getElementById(inputId);
    if (!input || input.disabled) return;

    if (direction > 0) {
        input.stepUp();
    } else {
        input.stepDown();
    }

    input.dispatchEvent(new Event('input', { bubbles: true }));
    input.dispatchEvent(new Event('change', { bubbles: true }));
    input.focus({ preventScroll: true });
};

/** Disables the remove button when only one row remains. */
function _updateRemoveButtons() {
    const rows = document.querySelectorAll('#dirList .dir-row');
    rows.forEach(row => {
        const btn = row.querySelector('.btn-remove-dir');
        if (btn) btn.disabled = rows.length <= 1;
    });
}

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
    if (currentMode === 'coverage') return;

    // 1. Gather data
    const dirInputs = document.querySelectorAll('#dirList .dir-row input');
    const directories = Array.from(dirInputs)
        .map(input => input.value.trim())
        .filter(v => v !== '');

    const params = {
        directories: directories,
        useCache: document.getElementById('keepMemory').checked,
        cacheDir: document.getElementById('cacheDir').value,
        resultsDir: document.getElementById('resultsDir').value,
        paranoidMode: document.getElementById('paranoidMode').checked,
        cpus: parseInt(document.getElementById('cpus').value) || 0,
        bufSize: parseInt(document.getElementById('bufSize').value) || 0,
        debugMode: document.getElementById('debugMode').checked,
    };

    // Clear old status/reset bar
    statusJob.textContent = "Starting up...";
    statusJob.classList.remove('status-value--success');
    statusFiles.textContent = "\u2014";
    statusDuplicates.textContent = "\u2014";
    statusDuplicates.classList.remove('status-value--orange');
    statusError.textContent = "";
    statusError.style.display = "none";
    resetProgressBar();

    // Hide previous results
    resultsSection.style.display = 'none';
    allGroups = [];
    currentPage = 1;
    window.currentPage = 1;
    resultsList.innerHTML = '';
    resultsPageSize.disabled = true;


    // UI State: Running
    setExecutionActive(true);
    stopButton.disabled = false;
    showResultsButton.disabled = true;
    clearResultsButton.disabled = true;
    fullResetButton.disabled = true;

    toggleStartSpinner(true);

    // 2. Call the Go backend function
    StartExecution(params)
        .then((result) => {
            if (result) statusJob.textContent = result;
            setExecutionActive(false);
            stopButton.disabled = true;
            fullResetButton.disabled = false;
            toggleStartSpinner(false);
        })
        .catch((err) => {
            statusJob.textContent = "Process Failed";
            statusJob.classList.remove('status-value--success');
            statusError.textContent = String(err);
            statusError.style.display = '';
            markActivePhaseFailed();
            showResultsButton.disabled = true;
            setExecutionActive(false);
            stopButton.disabled = true;
            fullResetButton.disabled = false;
            toggleStartSpinner(false);
        });
};

// --- Execution Cancellation Handler ---
window.cancelProcess = function () {
    statusJob.textContent = "Cancellation Requested...";

    // Disable the stop button immediately to prevent multiple presses
    stopButton.disabled = true;

    CancelExecution()
        .then(() => {
            statusJob.textContent = "Stopping process...";
        })
        .catch((err) => {
            statusJob.textContent = "Cancellation Error";
            statusError.textContent = `Failed to send cancellation: ${err}`;
            statusError.style.display = '';
            stopButton.disabled = false;
        });
};

window.showResults = function () {
    ShowResults()
        .catch((err) => {
            statusError.textContent = `Failed to open results file: ${err}`;
            statusError.style.display = '';
        });
};

// --- Full Reset Handler ---

/**
 * Resets all form inputs, results, and status UI back to their initial defaults.
 * Called both from window.fullReset() and from the backend "fullReset" Wails event.
 */
function _applyUIReset() {
    currentMode = 'duplicates';
    modeInputs.forEach(input => { input.checked = input.value === currentMode; });
    resetFolderSelections();

    // Reset advanced settings
    document.getElementById('cacheDir').value = '';
    document.getElementById('resultsDir').value = '';
    document.getElementById('cpus').value = '0';
    document.getElementById('bufSize').value = '1024';
    document.getElementById('paranoidMode').checked = false;
    document.getElementById('debugMode').checked = false;
    document.getElementById('keepMemory').checked = true;

    // Reset results panel and status area
    clearResults();

    // Restore button states
    setExecutionActive(false);
    stopButton.disabled = true;
    fullResetButton.disabled = false;
    toggleStartSpinner(false);
    updateModePresentation();
}

window.fullReset = function () {
    fullResetButton.disabled = true;

    // UI reset is driven by the "fullReset" Wails event emitted by the backend,
    // so the .then() only needs to handle unexpected binding-level errors.
    FullReset()
        .catch((err) => {
            statusError.textContent = `Full Reset failed: ${err}`;
            statusError.style.display = '';
            fullResetButton.disabled = false;
        });
};

window.clearResults = function () {
    // Reset in-memory state
    allGroups = [];
    currentPage = 1;
    window.currentPage = 1;

    // Clear results panel
    resultsList.innerHTML = '';
    resultsSection.style.display = 'none';
    resultsCountLabel.textContent = 'Results';
    clearResultsButton.disabled = true;
    resultsPageSize.disabled = true;
    showResultsButton.disabled = true;

    // Reset status area to clean slate
    statusJob.textContent = 'Ready to run.';
    statusJob.classList.remove('status-value--success');
    statusFiles.textContent = '\u2014';
    statusDuplicates.textContent = '\u2014';
    statusDuplicates.classList.remove('status-value--orange');
    coverageOutcome.textContent = 'Awaiting a coverage run';
    coverageCovered.textContent = '\u2014';
    coverageMissing.textContent = '\u2014';
    statusError.textContent = '';
    statusError.style.display = 'none';
    resetProgressBar();
    updateModePresentation();
};

function resetProgressBar() {
    activePhase = '';
    progressBar.style.width = '0%';
    progressBar.style.transform = '';
    progressBar.textContent = '';
    progressBar.classList.remove(
        'progress-bar--indeterminate',
        'progress-bar--success',
        'progress-bar--error',
    );
}

function updatePhaseProgress(title, percent) {
    if (title === 'Done' || title === 'Error' || !Number.isFinite(percent)) {
        return;
    }

    if (activePhase !== title) {
        resetProgressBar();
        activePhase = title;
    }

    const cappedPercent = Math.min(100, Math.max(0, percent));
    progressBar.classList.remove('progress-bar--error');

    if (cappedPercent === 0) {
        progressBar.style.width = '0%';
        progressBar.textContent = '';
        progressBar.classList.remove('progress-bar--success');
        progressBar.classList.add('progress-bar--indeterminate');
        return;
    }

    progressBar.classList.remove('progress-bar--indeterminate');
    progressBar.style.transform = '';
    progressBar.style.width = `${cappedPercent}%`;
    progressBar.textContent = cappedPercent > 5 ? `${cappedPercent.toFixed(2)}%` : '';
    progressBar.classList.toggle('progress-bar--success', cappedPercent >= 100);
}

function markActivePhaseFailed() {
    if (!activePhase) {
        return;
    }

    progressBar.classList.remove('progress-bar--indeterminate', 'progress-bar--success');
    progressBar.classList.add('progress-bar--error');
    progressBar.style.transform = '';
    progressBar.style.width = '100%';
    progressBar.textContent = 'Error';
}

// --- Status Listener Setup ---
function setupStatusListeners() {
    const showResultsButton = document.getElementById('showResultsButton'); // Get the element again
    // 1. Progress/Title Update Event
    runtime.EventsOn("progressUpdate", (data) => {
        const phaseChanged = statusJob.textContent !== data.title;
        if (phaseChanged) {
            statusJob.textContent = data.title;
            statusJob.classList.remove('status-value--success');
        }
        if (data.percent !== undefined) {
            const rawPercent = parseFloat(data.percent);
            updatePhaseProgress(data.title, rawPercent);
            showResultsButton.disabled = true;
        }
    });

    // 2. Detailed log / cancellation messages route to the job field
    runtime.EventsOn("detailedLog", (message) => {
        statusJob.textContent = message;
    });

    // 2a. Files count update
    runtime.EventsOn("filesCount", (data) => {
        if (data.total > 0) {
            statusFiles.textContent = `${data.current} of ${data.total}`;
        } else {
            statusFiles.textContent = `${data.current}`;
        }
    });

    // 3. Error Event
    runtime.EventsOn("errorUpdate", (message) => {
        statusJob.textContent = "Error: Process Failed";
        statusJob.classList.remove('status-value--success');
        statusError.textContent = message;
        statusError.style.display = '';
        markActivePhaseFailed();

        showResultsButton.disabled = true;
        fullResetButton.disabled = false;
        setExecutionActive(false);

        toggleStartSpinner(false);
    });

    runtime.EventsOn("executionFinished", (filePath) => {
    statusJob.textContent = "Process Complete.";
    statusJob.classList.add('status-value--success');
    toggleStartSpinner(false);
    setExecutionActive(false);
    stopButton.disabled = true;
    fullResetButton.disabled = false;

    // Fetch and display duplicate groups
    GetResults()
        .then(groups => renderResults(groups))
        .catch(err => console.error('GetResults error:', err));
});

    // fullReset event: backend notifies the frontend after FullReset() completes
    // (safety net — e.g. reset triggered mid-execution from an external source).
    // Calls _applyUIReset() directly to avoid re-invoking the backend.
    runtime.EventsOn("fullReset", () => {
        _applyUIReset();
    });
}
// Run setup after DOM load
setupStatusListeners();
refreshResultsButtonState();
updateModePresentation();

// --- Results: public page navigation (called from template onclick) ---
window.currentPage = currentPage; // expose for onclick expressions
window.goToPage = function (page) {
    renderPage(page);
};

window.changeResultsPageSize = function (value) {
    const nextPageSize = Number.parseInt(value, 10);
    if (!PAGE_SIZE_OPTIONS.includes(nextPageSize) || nextPageSize === pageSize) {
        resultsPageSize.value = String(pageSize);
        return;
    }

    const firstVisibleGroupIndex = (currentPage - 1) * pageSize;
    pageSize = nextPageSize;
    currentPage = Math.floor(firstVisibleGroupIndex / pageSize) + 1;
    window.currentPage = currentPage;

    if (allGroups.length > 0) {
        renderPage(currentPage, false);
    }
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
 * @param {boolean} scrollToResults whether to scroll the result section into view
 */
function renderPage(page, scrollToResults = true) {
    const totalPages = Math.max(1, Math.ceil(allGroups.length / pageSize));
    currentPage = Math.max(1, Math.min(page, totalPages));
    window.currentPage = currentPage;

    const start = (currentPage - 1) * pageSize;
    const end = Math.min(start + pageSize, allGroups.length);
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

    if (scrollToResults) {
        resultsSection.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
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

    const groupCount = allGroups.length;
    const totalDups = allGroups.reduce((sum, g) => sum + (g.duplicates ? g.duplicates.length : 0), 0);
    statusDuplicates.textContent = groupCount === 0 ? '0' : `${totalDups}`;
    statusDuplicates.classList.toggle('status-value--orange', totalDups > 0);

    if (groupCount === 0) {
        resultsCountLabel.textContent = 'Results';
        resultsSection.style.display = 'none';
        resultsPageSize.disabled = true;
        return;
    }

    resultsCountLabel.textContent =
        `Results — ${groupCount} group${groupCount !== 1 ? 's' : ''}, ${totalDups} duplicate${totalDups !== 1 ? 's' : ''}`;
    resultsSection.style.display = 'block';
    clearResultsButton.disabled = false;
    resultsPageSize.disabled = false;
    resultsPageSize.value = String(pageSize);
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
