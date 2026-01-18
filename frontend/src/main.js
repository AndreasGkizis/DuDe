import './style.css';
import htmlTemplate from './template.html?raw';

import { SelectFolder, StartExecution, ShowResults, CancelExecution, CheckIfResultsExist } from '../wailsjs/go/processing/FrontendApp';

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
});
}
// Run setup after DOM load
setupStatusListeners();
refreshResultsButtonState();

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