import assert from 'node:assert/strict';
import { setImmediate } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';
import test from 'node:test';
import { JSDOM } from 'jsdom';
import { createServer } from 'vite';

test('duplicate rendering restores buttons from the published result', async (t) => {
    const dom = new JSDOM('<div id="app"></div>');
    const listeners = new Map();
    let currentResult;
    globalThis.window = dom.window;
    globalThis.document = dom.window.document;
    globalThis.runtime = {
        EventsOn: (name, callback) => listeners.set(name, callback),
    };
    // JSDOM has no layout/scrolling implementation.
    dom.window.HTMLElement.prototype.scrollIntoView = () => {};
    dom.window.go = {
        processing: {
            FrontendApp: {
                CheckIfResultsExist: () => Promise.resolve(false),
                GetResults: () => Promise.resolve(currentResult),
                StartExecution: () => Promise.resolve(),
            },
        },
    };
    // Use Vite's existing loaders for CSS, raw HTML, and frontend ES modules.
    const server = await createServer({
        root: fileURLToPath(new URL('../', import.meta.url)),
        configFile: false,
        server: { middlewareMode: true, hmr: false },
    });
    t.after(async () => {
        await server.close();
        dom.window.close();
        delete globalThis.window;
        delete globalThis.document;
        delete globalThis.runtime;
    });
    await server.ssrLoadModule('/src/main.js');
    await setImmediate();

    const group = {
        FileName: 'a.txt',
        FilePath: '/data/a.txt',
        DuplicatesFound: [{ FileName: 'b.txt', FilePath: '/data/b.txt' }],
    };
    const diagnostic = {
        path: '/data/private',
        reason: 'permission denied',
        phase: 'walking',
        scope: 'duplicate',
    };
    const get = (id) => document.getElementById(id);

    for (const scenario of [
        {
            name: 'groups and a report',
            result: { groups: [group], reportPath: '/reports/duplicates.csv' },
            canOpen: true, canClear: true, groups: 1, diagnostics: false,
        },
        {
            name: 'diagnostics only and a report',
            result: { groups: [], diagnostics: { diagnostics: [diagnostic] }, reportPath: '/reports/skipped.csv' },
            canOpen: true, canClear: true, groups: 0, diagnostics: true,
        },
        {
            name: 'groups without a report',
            result: { groups: [group] },
            canOpen: false, canClear: true, groups: 1, diagnostics: false,
        },
        {
            name: 'diagnostics without a report',
            result: { diagnostics: { diagnostics: [diagnostic] } },
            canOpen: false, canClear: true, groups: 0, diagnostics: true,
        },
        {
            name: 'empty result',
            result: { groups: [], diagnostics: { diagnostics: [] }, reportPath: '' },
            canOpen: false, canClear: false, groups: 0, diagnostics: false,
        },
        {
            name: 'no result',
            result: null,
            canOpen: false, canClear: false, groups: 0, diagnostics: false,
        },
    ]) {
        await t.test(scenario.name, async () => {
            window.startProcess();
            await setImmediate();
            assert.equal(get('showResultsButton').disabled, true);
            assert.equal(get('clearResultsButton').disabled, true);

            currentResult = scenario.result;
            listeners.get('executionFinished')();
            await setImmediate();

            assert.equal(get('showResultsButton').disabled, !scenario.canOpen, 'report access');
            assert.equal(get('clearResultsButton').disabled, !scenario.canClear, 'clear results');
            assert.equal(get('results-section').style.display, scenario.groups ? 'block' : 'none', 'group list visibility');
            assert.equal(get('coverage-diagnostics').hidden, !scenario.diagnostics, 'diagnostic visibility');
            if (scenario.diagnostics) {
                assert.match(get('coverage-diagnostics').textContent, /permission denied/);
            }
            if (scenario.groups) {
                assert.equal(document.querySelectorAll('.result-card').length, scenario.groups);
            }

            window.clearResults();
            assert.equal(get('showResultsButton').disabled, true);
            assert.equal(get('clearResultsButton').disabled, true);
            assert.equal(get('coverage-diagnostics').hidden, true);
        });
    }
});
