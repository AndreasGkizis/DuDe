import './style.css';
import './app.css';

import {SelectFolder,StartExecution} from '../wailsjs/go/processing/FrontendApp';
document.querySelector('#app').innerHTML = `
    <h2 class="form-title">🚀 Execution Parameters</h2>

    <div class="form-container">
        <div class="input-group directory-group">
            <h3>Directory Settings</h3>
            
            <label for="sourceDir">Source Directory</label>
            <div class="dir-chooser">
                <input class="input dir-input" id="sourceDir" type="text" value="" readonly placeholder="Click to select Source Folder">
                <button class="btn btn-select" onclick="selectAndSetDir('sourceDir')">Select</button>
            </div>
            
            <label for="targetDir">Target Directory (Optional)</label>
            <div class="dir-chooser">
                <input class="input dir-input" id="targetDir" type="text" value="" readonly placeholder="Click to select Target Folder">
                <button class="btn btn-select" onclick="selectAndSetDir('targetDir')">Select</button>
            </div>
            
            <label for="cacheDir">Cache Directory</label>
            <div class="dir-chooser">
                <input class="input dir-input" id="cacheDir" type="text" value="" placeholder="e.g. /tmp/cache">
                <button class="btn btn-select" onclick="selectAndSetDir('cacheDir')">Select</button>
            </div>
        </div>

        <div class="input-group system-group">
            <h3>System Settings</h3>
            
            <label for="cpus">CPUs</label>
            <input class="input" id="cpus" type="number" value="4">
            
            <label for="bufSize">Buffer Size</label>
            <input class="input" id="bufSize" type="number" value="1024">
            
            <div class="paranoid-check-box">
                <label for="paranoidMode">Paranoid Mode</label>
                <input type="checkbox" id="paranoidMode" class="checkbox-input">
            </div>
            </div>
        </div>
    </div>

    <div class="execute-box">
        <button class="btn btn-execute" onclick="startProcess()">Start Execution</button>
    </div>
    
<div id="status-area" style="margin-top: 25px; max-width: 900px; margin-left: auto; margin-right: auto;">
        
        <div id="progress-title" class="progress-title">Ready to run.</div>
        
        <div class="progress-container">
            <div id="progress-bar" class="progress-bar" style="width: 0%;"></div>
        </div>

        <div id="detailed-status" class="detailed-status">
            Awaiting configuration and start command.
        </div>
    </div>`;


// Get the element to display the path
let folderPathOutput = document.getElementById("folder-path-output");

// Get status element
let executionStatus = document.getElementById("execution-status");

// Function to select and fill a specific directory input field
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



// 🚀 NEW ELEMENTS 🚀
let progressTitle = document.getElementById("progress-title");
let progressBar = document.getElementById("progress-bar");
let detailedStatus = document.getElementById("detailed-status");

// ---------------------------------------------------------------------
// 1. Setup Event Listeners for the Status Bar
// We will use two specific events from the backend:
// - 'progressUpdate': For setting the bar percentage and title
// - 'detailedLog': For continuous, detailed status messages
// ---------------------------------------------------------------------

function setupStatusListeners() {
    // 1. Progress/Title Update Event (For the bar and title)
    // The Go backend should send an object like: {title: "Scanning Files (30%)", percent: 30}
    runtime.EventsOn("progressUpdate", (data) => {
        if (data.title) {
            progressTitle.innerText = data.title;
        }
        if (data.percent !== undefined) {
            const percent = Math.min(100, Math.max(0, data.percent));
            progressBar.style.width = `${percent}%`;
            progressBar.innerText = percent > 5 ? `${percent}%` : '';
        }
    });

    // 2. Detailed Log Event (For the scrollable output box)
    runtime.EventsOn("detailedLog", (message) => {
        // Append the new message to the existing content
        detailedStatus.innerHTML += `<div>[${new Date().toLocaleTimeString()}] ${message}</div>`;
        // Auto-scroll to the bottom
        detailedStatus.scrollTop = detailedStatus.scrollHeight;
    });

    // 3. Error Event (Resets bar and shows error in detail box)
    runtime.EventsOn("errorUpdate", (message) => {
        progressTitle.innerText = "Error: Process Failed";
        detailedStatus.innerHTML += `<div style="color: #E57373; font-weight: bold;">[ERROR] ${message}</div>`;
        progressBar.style.width = '100%';
        progressBar.style.backgroundColor = '#E57373'; // Red
    });
}

// ---------------------------------------------------------------------
// 2. Modify startProcess to initialize the status area
// ---------------------------------------------------------------------

window.startProcess = function () {
    // 1. Gather data and call StartExecution (logic remains the same)
  const params = {
        // Change keys from PascalCase to camelCase
        sourceDir: document.getElementById('sourceDir').value,
        targetDir: document.getElementById('targetDir').value,
        cacheDir: document.getElementById('cacheDir').value,
        resultsDir: "", 
        paranoidMode: document.getElementById('paranoidMode').checked,
        cpus: parseInt(document.getElementById('cpus').value) || 0,
        bufSize: parseInt(document.getElementById('bufSize').value) || 0,
        dualFolderModeEnabled: false, 
    };
    console.log(params)    
    // Clear old status/reset bar
    progressTitle.innerText = "Starting up...";
    detailedStatus.innerHTML = "";
    progressBar.style.width = '0%';
    progressBar.style.backgroundColor = '#4CAF50'; // Reset color

    // 2. Call the Go backend function
    StartExecution(params)
        .then((result) => {
            // Success response from the StartExecution call (not the final process status)
            // You can log this to the detailed status area
            detailedStatus.innerHTML += `<div>${result}</div>`;
        })
        .catch((err) => {
            // Fatal binding error
            progressTitle.innerText = `FATAL BINDING ERROR`;
            detailedStatus.innerHTML = `<div style="color: #E57373;">${err}</div>`;
        });
};


// Call the setup function after your document.querySelector('#app').innerHTML block
setupStatusListeners();