import './style.css';

import { SelectFolder, StartExecution, ShowResults } from '../wailsjs/go/processing/FrontendApp';

// --- HTML Template Setup ---
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
                        <h3>Core Settings</h3>
                        
                        <!-- Source Directory -->
                        <label for="sourceDir">Source Directory (Mandatory)</label>
                        <div class="dir-chooser">
                            <input class="input dir-input" id="sourceDir" type="text" value="" readonly placeholder="Click to select Source Folder">
                            <button class="btn btn-select" onclick="selectAndSetDir('sourceDir')">Select</button>
                        </div>
                        
                        <!-- Target Directory -->
                        <label for="targetDir">Target Directory (Optional)</label>
                        <div class="dir-chooser">
                            <input class="input dir-input" id="targetDir" type="text" value="" readonly placeholder="Click to select Target Folder">
                            <button class="btn btn-select" onclick="selectAndSetDir('targetDir')">Select</button>
                        </div>
                    </div>

                    <!-- 2. ADVANCED SETTINGS SECTION (COLLAPSIBLE) -->
                    
                    <div id="advancedSection" class="advanced-section" data-collapsed="true">
                        <button id="advancedToggle" class="advanced-toggle" onclick="toggleAdvanced()">
                            <span>Advanced Settings</span>
                            <svg id="collapseIndicator" class="toggle-indicator" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                        </button>
                        
                        <div id="advancedContent" class="advanced-content">
                            
                            <div>
                                <label for="cacheDir">Cache Directory</label>
                                <div class="dir-chooser">
                                    <input class="input dir-input" id="cacheDir" type="text" value="" placeholder="/tmp/cache">
                                    <button class="btn btn-select" onclick="selectAndSetDir('cacheDir')">Select</button>
                                </div>
                            </div>

                            <div>
                                <label for="resultsDir">Results File Path</label>
                                <div class="dir-chooser">
                                    <input class="input dir-input" id="resultsDir" type="text" value="" placeholder="/tmp/results.json">
                                    <button class="btn btn-select" onclick="selectAndSetDir('resultsDir')">Select</button>
                                </div>
                            </div>

                            <div class="full-width-item stacked-inputs">
                                
                                <div>
                                    <label for="cpus">CPUs</label>
                                    <input class="input" id="cpus" type="number" value="4">
                                </div>

                                <div>
                                    <label for="bufSize">Buffer Size</label>
                                    <input class="input" id="bufSize" type="number" value="1024">
                                </div>
                            </div>
                            
                            <div class="full-width-item checkbox-container">
                                <input type="checkbox" id="paranoidMode" class="checkbox-input">
                                <label for="paranoidMode">Paranoid Mode (Strict Validation)</label>
                            </div>
                        </div>
                    </div>
                    
                    <!-- EXECUTION BUTTON -->
                    <div class="execute-box">
                        <button class="btn btn-execute" onclick="startProcess()">
                            Start
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

        // Temporarily ensure padding is set for accurate scrollHeight measurement
        content.style.paddingTop = '15px';
        content.style.paddingBottom = '15px';

        const contentHeight = content.scrollHeight;

        // Set max-height large enough to ensure full transition (scrollHeight + buffer)
        content.style.maxHeight = (contentHeight + 50) + 'px';

    } else {
        // 1. Get current height before starting transition (ensures smooth collapse)
        content.style.maxHeight = content.scrollHeight + 'px';

        // 2. Force reflow
        void content.offsetWidth;

        // 3. Start collapse transition
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
        cacheDir: document.getElementById('cacheDir').value,
        resultsDir: document.getElementById('resultsDir').value,
        paranoidMode: document.getElementById('paranoidMode').checked,
        cpus: parseInt(document.getElementById('cpus').value) || 0,
        bufSize: parseInt(document.getElementById('bufSize').value) || 0,
        dualFolderModeEnabled: false,
    };
    console.log("Starting process with parameters:", params);

    // Clear old status/reset bar
    progressTitle.innerText = "Starting up...";
    detailedStatus.innerHTML = "";
    progressBar.style.width = '0%';
    // Reset color to the primary accent color (bright green)
    progressBar.style.backgroundColor = 'var(--color-accent)';


    // Hide button at start of process
    showResultsButton.disabled = true;
    // 2. Call the Go backend function
    StartExecution(params)
        .then((result) => {
            detailedStatus.innerHTML += `<div>${result}</div>`;
        })
        .catch((err) => {
            progressTitle.innerText = `FATAL BINDING ERROR`;
            detailedStatus.innerHTML = `<div style="color: #E57373;">${err}</div>`;
        });
};

window.showResults = function () {
    ShowResults()
        .catch((err) => {
            detailedStatus.innerHTML += `<div style="color: #E57373; font-weight: bold;">[ERROR] Failed to open results file: ${err}</div>`;
        });
};


// --- Status Listener Setup (Updated) ---
function setupStatusListeners() {
    const showResultsButton = document.getElementById('showResultsButton'); // Get the element again
    // 1. Progress/Title Update Event
    runtime.EventsOn("progressUpdate", (data) => {
        // ... existing title and percent logic ...
        progressBar.innerText = data.title
        if (data.percent !== undefined) {
            const percent = Math.min(100, Math.max(0, data.percent));
            progressBar.style.width = `${percent}%`;
            progressBar.innerText = percent > 5 ? `${percent}%` : '';

            if (percent >= 100) {
                progressTitle.innerText = "Process Complete.";
                showResultsButton.disabled = false;
            } else {
                showResultsButton.disabled = true;  
            }
        }
    });

    // 2. Detailed Log Event (unchanged)
    runtime.EventsOn("detailedLog", (message) => {
        detailedStatus.innerHTML += `<div>[${new Date().toLocaleTimeString()}] ${message}</div>`;
        detailedStatus.scrollTop = detailedStatus.scrollHeight;
    });

    // 3. Error Event (Updated)
    runtime.EventsOn("errorUpdate", (message) => {
        progressTitle.innerText = "Error: Process Failed";
        detailedStatus.innerHTML += `<div style="color: #E57373; font-weight: bold;">[ERROR] ${message}</div>`;
        progressBar.style.width = '100%';
        progressBar.style.backgroundColor = '#ff6b6b';

        showResultsButton.disabled = true;
    });
}
// Run setup after DOM load
setupStatusListeners();