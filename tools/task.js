#!/usr/bin/env node

/**
 * BramandScript Task & Todo CLI Tool
 * Tool-agnostic CLI for AI Agents and Developers to inspect, check, uncheck, and update tasks.
 * Zero external npm dependencies.
 */

const fs = require('fs');
const path = require('path');

// ANSI Colors
const RESET = '\x1b[0m';
const BOLD = '\x1b[1m';
const DIM = '\x1b[2m';
const RED = '\x1b[31m';
const GREEN = '\x1b[32m';
const YELLOW = '\x1b[33m';
const CYAN = '\x1b[36m';

function getActiveTaskFile() {
  const dateDir = path.join(process.cwd(), 'date');
  if (!fs.existsSync(dateDir)) {
    console.error(`${RED}[ERROR] 'date/' directory not found in ${process.cwd()}${RESET}`);
    process.exit(1);
  }

  const months = fs.readdirSync(dateDir).filter(f => fs.statSync(path.join(dateDir, f)).isDirectory());
  if (months.length === 0) {
    console.error(`${RED}[ERROR] No month directory found inside 'date/'${RESET}`);
    process.exit(1);
  }

  // Sort descending to get the latest month
  months.sort().reverse();
  const latestMonth = months[0];

  const targetFile = path.join(dateDir, latestMonth, 'First_week_task.md');
  if (!fs.existsSync(targetFile)) {
    console.error(`${RED}[ERROR] Task file not found: ${targetFile}${RESET}`);
    process.exit(1);
  }

  return { filePath: targetFile, month: latestMonth };
}

function parseTasks(content) {
  const lines = content.split('\n');
  const tasks = [];
  let currentDay = 'General';

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const dayMatch = line.match(/^##\s*(Day\s*\d+[^:]*):?/i);
    if (dayMatch) {
      currentDay = dayMatch[1].trim();
      continue;
    }

    const taskMatch = line.match(/^-\s*\[([ xX])\]\s*(.+)$/);
    if (taskMatch) {
      const isDone = taskMatch[1].toLowerCase() === 'x';
      const text = taskMatch[2].trim();
      tasks.push({
        index: tasks.length + 1,
        lineIndex: i,
        day: currentDay,
        done: isDone,
        text: text
      });
    }
  }

  return { lines, tasks };
}

function cmdList(filterDay) {
  const { filePath, month } = getActiveTaskFile();
  const content = fs.readFileSync(filePath, 'utf8');
  const { tasks } = parseTasks(content);

  console.log(`\n${BOLD}${CYAN}✦ BramandScript Task Dashboard (Month: ${month})${RESET}`);
  console.log(`${DIM}File: ${filePath}${RESET}\n`);

  let currentDay = '';
  let completed = 0;

  tasks.forEach(t => {
    if (filterDay && !t.day.toLowerCase().includes(filterDay.toLowerCase())) {
      return;
    }

    if (t.day !== currentDay) {
      currentDay = t.day;
      console.log(`\n${BOLD}${currentDay}${RESET}`);
      console.log(`${DIM}${'─'.repeat(45)}${RESET}`);
    }

    if (t.done) completed++;

    const statusBadge = t.done 
      ? `${GREEN}[✔ COMPLETE]${RESET}` 
      : `${YELLOW}[· PENDING ]${RESET}`;
    const num = String(t.index).padStart(2, '0');
    console.log(`  ${DIM}${num}.${RESET} ${statusBadge} ${t.done ? DIM + t.text + RESET : t.text}`);
  });

  const total = tasks.length;
  const pct = total > 0 ? Math.round((completed / total) * 100) : 0;
  console.log(`\n${DIM}${'═'.repeat(45)}${RESET}`);
  console.log(`${BOLD}Progress:${RESET} ${completed}/${total} tasks completed (${pct}%)`);
  printProgressBar(completed, total);
  console.log();
}

function printProgressBar(done, total) {
  const width = 30;
  const filled = total > 0 ? Math.round((done / total) * width) : 0;
  const empty = width - filled;
  const bar = `${GREEN}${'█'.repeat(filled)}${RESET}${DIM}${'░'.repeat(empty)}${RESET}`;
  console.log(`[${bar}]`);
}

function cmdCheck(target, markDone = true) {
  const { filePath } = getActiveTaskFile();
  const content = fs.readFileSync(filePath, 'utf8');
  const { lines, tasks } = parseTasks(content);

  let targetTask = null;
  const targetNum = parseInt(target, 10);

  if (!isNaN(targetNum)) {
    targetTask = tasks.find(t => t.index === targetNum);
  } else {
    const q = target.toLowerCase();
    targetTask = tasks.find(t => t.text.toLowerCase().includes(q));
  }

  if (!targetTask) {
    console.error(`${RED}[ERROR] No task found matching query: "${target}"${RESET}`);
    process.exit(1);
  }

  const mark = markDone ? 'x' : ' ';
  lines[targetTask.lineIndex] = `- [${mark}] ${targetTask.text}`;

  fs.writeFileSync(filePath, lines.join('\n'), 'utf8');
  const action = markDone ? `${GREEN}checked [x]${RESET}` : `${YELLOW}unmarked [ ]${RESET}`;
  console.log(`${BOLD}[OK]${RESET} Task #${targetTask.index} ("${targetTask.text}") ${action}`);
}

function cmdEdit(target, newText) {
  const { filePath } = getActiveTaskFile();
  const content = fs.readFileSync(filePath, 'utf8');
  const { lines, tasks } = parseTasks(content);

  let targetTask = null;
  const targetNum = parseInt(target, 10);

  if (!isNaN(targetNum)) {
    targetTask = tasks.find(t => t.index === targetNum);
  } else {
    const q = target.toLowerCase();
    targetTask = tasks.find(t => t.text.toLowerCase().includes(q));
  }

  if (!targetTask) {
    console.error(`${RED}[ERROR] No task found matching query: "${target}"${RESET}`);
    process.exit(1);
  }

  const mark = targetTask.done ? 'x' : ' ';
  lines[targetTask.lineIndex] = `- [${mark}] ${newText}`;

  fs.writeFileSync(filePath, lines.join('\n'), 'utf8');
  console.log(`${BOLD}${GREEN}[OK]${RESET} Task #${targetTask.index} updated: "${newText}"`);
}

function cmdView(target) {
  const { filePath } = getActiveTaskFile();
  const content = fs.readFileSync(filePath, 'utf8');
  const { tasks } = parseTasks(content);

  let targetTask = null;
  const targetNum = parseInt(target, 10);

  if (!isNaN(targetNum)) {
    targetTask = tasks.find(t => t.index === targetNum);
  } else {
    const q = target.toLowerCase();
    targetTask = tasks.find(t => t.text.toLowerCase().includes(q));
  }

  if (!targetTask) {
    console.error(`${RED}[ERROR] No task found matching query: "${target}"${RESET}`);
    process.exit(1);
  }

  console.log(`\n${BOLD}${CYAN}✦ Task Details — #${targetTask.index}${RESET}`);
  console.log(`${BOLD}Day:${RESET}         ${targetTask.day}`);
  console.log(`${BOLD}Status:${RESET}      ${targetTask.done ? GREEN + '[✔ COMPLETE]' + RESET : YELLOW + '[· PENDING ]' + RESET}`);
  console.log(`${BOLD}Specification:${RESET} ${targetTask.text}\n`);
}

function parseFlags(rawArgs) {
  const flags = {
    _: [],
    day: null,
    desc: null,
    includes: null,
  };

  for (let i = 0; i < rawArgs.length; i++) {
    const arg = rawArgs[i];
    if (arg.startsWith('--day=')) {
      flags.day = arg.slice(6);
    } else if (arg === '--day' && i + 1 < rawArgs.length) {
      flags.day = rawArgs[++i];
    } else if (arg.startsWith('--desc=')) {
      flags.desc = arg.slice(7);
    } else if (arg === '--desc' && i + 1 < rawArgs.length) {
      flags.desc = rawArgs[++i];
    } else if (arg.startsWith('--description=')) {
      flags.desc = arg.slice(14);
    } else if (arg === '--description' && i + 1 < rawArgs.length) {
      flags.desc = rawArgs[++i];
    } else if (arg.startsWith('--includes=')) {
      flags.includes = arg.slice(11);
    } else if (arg === '--includes' && i + 1 < rawArgs.length) {
      flags.includes = rawArgs[++i];
    } else if (arg.startsWith('--include=')) {
      flags.includes = arg.slice(10);
    } else if (arg === '--include' && i + 1 < rawArgs.length) {
      flags.includes = rawArgs[++i];
    } else if (!arg.startsWith('--')) {
      flags._.push(arg);
    }
  }

  return flags;
}

function cmdAdd(title, dayArg, descArg, includesArg) {
  if (!title || !title.trim()) {
    console.error(`${RED}[ERROR] Task title cannot be empty!${RESET}`);
    console.error(`${YELLOW}Usage: bmd-task add "Task title" --desc="Detailed description" [--includes="..."] [--day=N]${RESET}`);
    process.exit(1);
  }

  if (!descArg || descArg.trim().length < 15) {
    console.error(`\n${RED}${BOLD}✦ [ERROR] Compulsory Requirement Missing: Detailed Description Required!${RESET}`);
    console.error(`${YELLOW}Every new task MUST include a detailed description and scope specification (minimum 15 characters).${RESET}\n`);
    console.error(`${BOLD}Required Command Syntax:${RESET}`);
    console.error(`  bmd-task add "Task Title" --desc="Detailed explanation of the task" [--includes="spec1, spec2, ..."] [--day=N]\n`);
    console.error(`${BOLD}Flags:${RESET}`);
    console.error(`  ${CYAN}--desc="..."${RESET}       : ${BOLD}[COMPULSORY]${RESET} Detailed description of what must be built.`);
    console.error(`  ${CYAN}--includes="..."${RESET}   : ${DIM}[RECOMMENDED]${RESET} Specific deliverables, components, or test scopes.`);
    console.error(`  ${CYAN}--day=N${RESET}            : Target day (e.g., --day=4).\n`);
    console.error(`${BOLD}Example:${RESET}`);
    console.error(`  bmd-task add "Mutex Synchronization" --day=4 \\`);
    console.error(`    --desc="Thread-safe mutual exclusion primitive with lock and unlock methods" \\`);
    console.error(`    --includes="Mutex object in object/mutex.go, OpLock/OpUnlock opcodes, race test verification"\n`);
    process.exit(1);
  }

  const { filePath } = getActiveTaskFile();
  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split('\n');

  const targetDay = dayArg ? `Day ${dayArg}` : 'Day 1';
  let insertIndex = -1;

  for (let i = 0; i < lines.length; i++) {
    if (lines[i].includes(`## ${targetDay}`)) {
      // Find the end of this day's task list (before the next '##' or '---')
      insertIndex = i + 1;
      while (insertIndex < lines.length && !lines[insertIndex].startsWith('##') && !lines[insertIndex].startsWith('---')) {
        insertIndex++;
      }
      break;
    }
  }

  let formattedTask = title.trim();
  const cleanDesc = descArg.trim();
  
  if (formattedTask.includes(':')) {
    formattedTask = `${formattedTask} — ${cleanDesc}`;
  } else {
    formattedTask = `${formattedTask}: ${cleanDesc}`;
  }

  if (includesArg && includesArg.trim()) {
    formattedTask = `${formattedTask} (Includes: ${includesArg.trim()})`;
  }

  if (insertIndex === -1) {
    lines.push(`\n## ${targetDay}`);
    lines.push(`- [ ] ${formattedTask}`);
  } else {
    lines.splice(insertIndex, 0, `- [ ] ${formattedTask}`);
  }

  fs.writeFileSync(filePath, lines.join('\n'), 'utf8');
  console.log(`${BOLD}${GREEN}[OK]${RESET} Task added to ${targetDay}:`);
  console.log(`  ${BOLD}Title:${RESET}         ${title.trim()}`);
  console.log(`  ${BOLD}Description:${RESET}   ${cleanDesc}`);
  if (includesArg && includesArg.trim()) {
    console.log(`  ${BOLD}Includes:${RESET}      ${includesArg.trim()}`);
  }
}

function cmdInit(mmyy) {
  const now = new Date();
  const defaultMM = String(now.getMonth() + 1).padStart(2, '0');
  const defaultYY = String(now.getFullYear()).slice(-2);
  const targetMMYY = mmyy || `${defaultMM}${defaultYY}`;

  const targetDir = path.join(process.cwd(), 'date', targetMMYY);
  if (!fs.existsSync(targetDir)) {
    fs.mkdirSync(targetDir, { recursive: true });
  }

  const targetFile = path.join(targetDir, 'First_week_task.md');
  if (fs.existsSync(targetFile)) {
    console.log(`${YELLOW}[WARN] Task file already exists: ${targetFile}${RESET}`);
    return;
  }

  const template = `# BramandScript — Month ${targetMMYY}: First Week Task\n\n## Day 1: Foundation\n- [ ] Initialize project\n`;
  fs.writeFileSync(targetFile, template, 'utf8');
  console.log(`${BOLD}${GREEN}[OK]${RESET} Initialized new tracking directory: ${targetFile}`);
}

// ---------------- CLI Dispatcher ----------------
const args = process.argv.slice(2);
const command = args[0] || 'list';

switch (command) {
  case 'list':
  case 'ls':
    cmdList(args[1]);
    break;

  case 'view':
  case 'show':
  case 'info':
    if (!args[1]) {
      console.error(`${RED}Usage: bmd-task view <task-number | "search text">${RESET}`);
      process.exit(1);
    }
    cmdView(args.slice(1).join(' '));
    break;

  case 'check':
  case 'done':
  case 'complete':
    if (!args[1]) {
      console.error(`${RED}Usage: bmd-task check <task-number | "search text">${RESET}`);
      process.exit(1);
    }
    cmdCheck(args.slice(1).join(' '), true);
    break;

  case 'uncheck':
  case 'undo':
    if (!args[1]) {
      console.error(`${RED}Usage: bmd-task uncheck <task-number | "search text">${RESET}`);
      process.exit(1);
    }
    cmdCheck(args.slice(1).join(' '), false);
    break;

  case 'edit':
  case 'update':
    if (!args[1] || !args[2]) {
      console.error(`${RED}Usage: bmd-task edit <task-number | "search text"> "New Task Text"${RESET}`);
      process.exit(1);
    }
    cmdEdit(args[1], args.slice(2).join(' '));
    break;

  case 'add':
    const flags = parseFlags(args.slice(1));
    const titleArg = flags._.join(' ');
    cmdAdd(titleArg, flags.day, flags.desc, flags.includes);
    break;

  case 'init':
    cmdInit(args[1]);
    break;

  case 'help':
  case '--help':
  case '-h':
    console.log(`
${BOLD}✦ BramandScript Task & Todo CLI Tool${RESET}

${BOLD}Commands:${RESET}
  bmd-task list [day]                                  List tasks (optional filter by day)
  bmd-task view <id | text>                            View task full description & details
  bmd-task check <id | text>                           Mark task as complete [x]
  bmd-task uncheck <id | text>                         Mark task as pending [ ]
  bmd-task edit <id | text> "New Task Text"            Update existing task text
  bmd-task add "Title" --desc="..." [--day=N]          Add new task (${BOLD}detailed --desc is COMPULSORY${RESET})
  bmd-task init [MMYY]                                 Initialize monthly tracking folder

${BOLD}Flags for 'add' command:${RESET}
  ${CYAN}--desc="..."${RESET}       ${BOLD}[COMPULSORY]${RESET} Detailed description of what must be implemented.
  ${CYAN}--includes="..."${RESET}   ${DIM}[RECOMMENDED]${RESET} Deliverables, components, or test scopes.
  ${CYAN}--day=N${RESET}            Target day number (e.g., --day=4). Defaults to Day 1.
`);
    break;

  default:
    console.error(`${RED}Unknown command: ${command}${RESET}. Run 'bmd-task help' for usage.`);
    process.exit(1);
}
