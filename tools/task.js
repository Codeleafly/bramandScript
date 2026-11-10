#!/usr/bin/env node

/**
 * BramandScript Task & Todo CLI Tool (Legacy Node.js Compatibility Wrapper)
 * Dispatches to 100% Native BramandScript engine (tools/task.bmd).
 */

const { spawnSync } = require('child_process');
const path = require('path');

const repoDir = path.join(__dirname, '..');
const bmdExe = path.join(repoDir, 'bramand');
const taskBmd = path.join(repoDir, 'tools', 'task.bmd');

const res = spawnSync(bmdExe, ['run', taskBmd, ...process.argv.slice(2)], {
  stdio: 'inherit'
});

process.exit(res.status !== null ? res.status : 0);
