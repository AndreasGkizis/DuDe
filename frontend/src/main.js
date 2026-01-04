import './style.css';

import { SelectFolder, StartExecution, ShowResults, CancelExecution } from '../wailsjs/go/processing/FrontendApp';

document.querySelector('#app').innerHTML = `
            <div class="main-content-wrapper">
                <h2 class="app-title">
                    <pre class="ascii-art">
██████╗  ██╗   ██╗        ██████╗  ███████╗ 
██╔══██╗ ██║   ██║        ██╔══██╗ ██╔════╝ 
██║  ██║ ██║   ██║ █████╗ ██║  ██║ █████╗   
██║  ██║ ██║   ██║ ╚════╝ ██║  ██║ ██╔══╝   
██████╔╝ ╚██████╔╝        ██████╔╝ ███████╗ 
╚═════╝   ╚═════╝         ╚═════╝  ╚══════╝ 
--------------------------------------------
    Welcome to Duplicate Detection      
--------------------------------------------
</pre>
                </h2>

                <div class="main-container">

                    <!-- 1. CORE CONFIGURATION SECTION (Directories) -->
                    <div class="input-group mb-6">
                        <h3>Settings</h3>
                        
                        <label for="sourceDir">Source Directory (Mandatory)
                        <span class="tooltip-container">
                                <span class="info-icon">i</span>
                                <span class="tooltip-text">The **primary folder** where files will be scanned for duplicates.</span>
                            </span></label>
                        <div class="dir-chooser">
                            <input class="input dir-input" id="sourceDir" type="text" value="" readonly placeholder="Nothing selected!  use this button--->">
                            <button class="btn btn-select" onclick="selectAndSetDir('sourceDir')">Select</button>
                        </div>
                        
                        <label for="targetDir">Target Directory (Optional)
                        
                        <span class="tooltip-container">
                                <span class="info-icon">i</span>
                                <span class="tooltip-text">An **additional folder** to scan for duplicates. If used, results will only show duplicates found between Source and Target.</span>
                            </span>
                            </label>
                        <div class="dir-chooser">
                            <input class="input dir-input" id="targetDir" type="text" value="" readonly placeholder="Nothing selected!  use this button--->">
                            <button class="btn btn-select" onclick="selectAndSetDir('targetDir')">Select</button>
                        </div>
                    </div>

                    <!-- 2. ADVANCED SETTINGS SECTION -->
                    
                    <div id="advancedSection" class="advanced-section" data-collapsed="true">
                        <button id="advancedToggle" class="advanced-toggle" onclick="toggleAdvanced()">
                            <span>Advanced Settings</span>
                            <svg id="collapseIndicator" class="toggle-indicator" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                        </button>
                        
                        <div id="advancedContent" class="advanced-content">
                            
                            <div>
                                <label for="cacheDir">Cache Directory
                                
                                <span class="tooltip-container tooltip-bottom-left">
                                        <span class="info-icon">i</span>
                                        <span class="tooltip-text ">Location to store pre-calculated file hashes. Reusing hashes speeds up subsequent runs. Defaults to OS temp folder.</span>
                                    </span>
                                    </label>
                                <div class="dir-chooser">
                                    <input class="input dir-input" id="cacheDir" type="text" value="" placeholder="/tmp/cache">
                                    <button class="btn btn-select" onclick="selectAndSetDir('cacheDir')">Select</button>
                                </div>
                            </div>

                            <div>
                                <label for="resultsDir">Results File Path
                                
                                <span class="tooltip-container tooltip-bottom-left">
                                        <span class="info-icon">i</span>
                                        <span class="tooltip-text">The full path where the final JSON report will be saved. Defaults to OS temp folder.</span>
                                    </span>
                                    </label>
                                <div class="dir-chooser">
                                    <input class="input dir-input" id="resultsDir" type="text" value="" placeholder="/tmp/results.json">
                                    <button class="btn btn-select" onclick="selectAndSetDir('resultsDir')">Select</button>
                                </div>
                            </div>

                            <div class="full-width-item stacked-inputs">
                                
                                <div>
                                    <label for="cpus">CPUs
                                    
                                    <span class="tooltip-container">
                                            <span class="info-icon">i</span>
                                            <span class="tooltip-text">Number of CPU cores to use for parallel hashing. 0 means all available cores.</span>
                                        </span>
                                        </label>
                                    <input class="input" id="cpus" type="number" value="8">
                                </div>

                                <div>
                                    <label for="bufSize">Buffer Size
                                    <span class="tooltip-container">
                                            <span class="info-icon">i</span>
                                            <span class="tooltip-text">I/O buffer size (in KB) when reading files for hashing. Larger size can improve speed on HDDs.</span>
                                        </span>
                                        </label>
                                    <input class="input" id="bufSize" type="number" value="1024">
                                </div>
                            </div>

                            <div class="full-width-item checkbox-container">
                                <input type="checkbox" id="paranoidMode" class="checkbox-input">
                                <label for="paranoidMode">
                                    Paranoid Mode (Strict Validation)
                                    <span class="tooltip-container tooltip-top">
                                        <span class="info-icon">i</span>
                                        <span class="tooltip-text">Performs additional file size/timestamp checks before hashing to eliminate false positives. Slower!</span>
                                    </span>
                                </label>
                            </div>                           
                            
                            <div class="full-width-item checkbox-container">
                                <input type="checkbox" id="keepLogs" class="checkbox-input">
                                <label for="keepLogs">
                                    Keep Logs
                                    <span class="tooltip-container tooltip-top">
                                        <span class="info-icon">i</span>
                                        <span class="tooltip-text">Decides on whether to keep logs of the execution or not (ON by default)</span>
                                    </span>
                                </label>
                            </div>                           
                            
                            <div class="full-width-item checkbox-container">
                                <input type="checkbox" id="keepMemory" class="checkbox-input">
                                <label for="keepMemory">
                                    Keep Memory of run
                                    <span class="tooltip-container tooltip-top">
                                        <span class="info-icon">i</span>
                                        <span class="tooltip-text">Decides on whether to keep a memory of the run in order to not redo work in case of looking up in the same folders (ON by default, creates a memory.db file which can be safely deleted)</span>
                                    </span>
                                </label>
                            </div>
   
                        </div>
                    </div>
                    
                    <!-- EXECUTION BUTTON -->
                    <div class="execute-box">
                        <button id="startButton" class="btn btn-execute" onclick="startProcess()">
                                <span id="startText">Start</span>
                                <span id="startButtonSpinner" class="spinner-hidden"></span>
                            </button>
                        <button id="stopButton" class="btn btn-stop" onclick="cancelProcess()" disabled>
                            Stop
                        </button>
                    </div>

                    <div class="results-button-container">
                        <button id="showResultsButton" class="btn btn-show-results" onclick="showResults()" disabled>
                            Open Results File
                        </button>
                    </div>

                </div>

                <!-- STATUS AREA -->
                <div id="status-area">
                    <div id="progress-title" class="progress-title">Ready to run.</div>
                    
                    <div class="progress-container">
                        <div id="progress-bar" class="progress-bar" style="width: 0%;"></div>
                    </div>

                    <div id="detailed-status" class="detailed-status">
                        Awaiting configuration and start command.
                    </div>
                </div>
                
                <footer class="app-footer">Made by Andreas with <3</footer>
            </div>
        `;

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
/**
 * Toggles the advanced settings section's collapsed state using max-height transition.
 */
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
        useCache: document.getElementById('keepMemory').value,
        cacheDir: document.getElementById('cacheDir').value,
        resultsDir: document.getElementById('resultsDir').value,
        paranoidMode: document.getElementById('paranoidMode').checked,
        cpus: parseInt(document.getElementById('cpus').value) || 0,
        bufSize: parseInt(document.getElementById('bufSize').value) || 0,
        keepLogs: document.getElementById('keepMemory').checked,
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
                showResultsButton.disabled = false;
                toggleStartSpinner(false);
            } else {
                showResultsButton.disabled = true;
            }
        }
    });

    // 2. Detailed Log Event (unchanged)
    runtime.EventsOn("detailedLog", (message) => {

        const tolerance = 20;
        const isScrolledToBottom = (detailedStatus.scrollHeight - detailedStatus.clientHeight) <= (detailedStatus.scrollTop + tolerance);

        const logEntry = document.createElement('div');
        logEntry.innerHTML = `[${new Date().toLocaleTimeString()}] ${message}`;

        // Use appendChild instead of innerHTML +=
        detailedStatus.appendChild(logEntry);

        while (detailedStatus.children.length > MAX_LOG_ROWS) {
            // detailedStatus.firstChild is the oldest element appended
            detailedStatus.removeChild(detailedStatus.firstChild);
        }

        // 4. SCROLL: Only scroll to bottom if the user was already at the bottom
        if (isScrolledToBottom) {
            detailedStatus.scrollTop = detailedStatus.scrollHeight;
        }

    });

    // 3. Error Event (Updated)
    runtime.EventsOn("errorUpdate", (message) => {
        progressTitle.innerText = "Error: Process Failed";
        detailedStatus.innerHTML += `<div style="color: #E57373; font-weight: bold;">[ERROR] ${message}</div>`;
        progressBar.style.width = '100%';
        progressBar.style.backgroundColor = '#ff6b6b';

        showResultsButton.disabled = true;

        toggleStartSpinner(false);
    });
}
// Run setup after DOM load
setupStatusListeners();

// --- Spinner State Handler ---
/**
 * Toggles the visibility of the start button text and spinner.
 * @param {boolean} isRunning - True if the process is starting/running.
 */
function toggleStartSpinner(isLoading) {
    if (isLoading) {
        startButton.classList.add('is-loading');
        // If you had content:'◢' here for the character spinner, remove it, 
        // as the dots spinner relies on the CSS pseudo-elements.
        startButtonSpinner.textContent = ''; 
    } else {
        startButton.classList.remove('is-loading');
        startButtonSpinner.textContent = '';
    }
}