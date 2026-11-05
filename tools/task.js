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

function getActiveTaskFile(weekArg, autoCreate = false) {
  const dateDir = path.join(process.cwd(), 'date');
  if (!fs.existsSync(dateDir)) {
    if (autoCreate) {
      fs.mkdirSync(dateDir, { recursive: true });
    } else {
      console.error(`${RED}[ERROR] 'date/' directory not found in ${process.cwd()}${RESET}`);
      process.exit(1);
    }
  }

  let months = fs.existsSync(dateDir) ? fs.readdirSync(dateDir).filter(f => fs.statSync(path.join(dateDir, f)).isDirectory()) : [];
  if (months.length === 0) {
    if (autoCreate) {
      const now = new Date();
      const mm = String(now.getMonth() + 1).padStart(2, '0');
      const yy = String(now.getFullYear()).slice(-2);
      const newMonth = `${mm}${yy}`;
      fs.mkdirSync(path.join(dateDir, newMonth), { recursive: true });
      months = [newMonth];
    } else {
      console.error(`${RED}[ERROR] No month directory found inside 'date/'${RESET}`);
      process.exit(1);
    }
  }

  // Sort descending to get the latest month
  months.sort().reverse();
  const latestMonth = months[0];
  const monthDir = path.join(dateDir, latestMonth);

  const weekNames = ['First', 'Second', 'Third', 'Fourth', 'Fifth'];
  let targetFile = null;

  // Extract --week flag from process.argv if not passed directly
  if (!weekArg) {
    for (const arg of process.argv) {
      if (arg.startsWith('--week=')) {
        weekArg = arg.split('=')[1];
        break;
      }
    }
  }

  if (weekArg) {
    const num = parseInt(weekArg, 10);
    if (!isNaN(num) && num >= 1 && num <= 5) {
      targetFile = path.join(monthDir, `${weekNames[num - 1]}_week_task.md`);
      if (!fs.existsSync(targetFile) && autoCreate) {
        cmdInitWeek(num, latestMonth);
      }
    } else {
      const cap = weekArg.charAt(0).toUpperCase() + weekArg.slice(1).toLowerCase();
      targetFile = path.join(monthDir, `${cap}_week_task.md`);
    }
  } else {
    // Find the latest existing week task file or default to First_week_task.md
    for (let i = weekNames.length - 1; i >= 0; i--) {
      const cand = path.join(monthDir, `${weekNames[i]}_week_task.md`);
      if (fs.existsSync(cand)) {
        targetFile = cand;
        break;
      }
    }
    if (!targetFile) {
      targetFile = path.join(monthDir, 'First_week_task.md');
      if (!fs.existsSync(targetFile) && autoCreate) {
        cmdInitWeek(1, latestMonth);
      }
    }
  }

  if (!fs.existsSync(targetFile)) {
    console.error(`${RED}[ERROR] Task file not found: ${targetFile}${RESET}`);
    console.error(`${YELLOW}[HINT] You can initialize it with: bmd-task init-week <week_number>${RESET}`);
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

function cmdList(rawArgs) {
  let flags = { _: [], day: null, week: null };
  if (Array.isArray(rawArgs)) {
    flags = parseFlags(rawArgs);
  } else if (typeof rawArgs === 'string') {
    flags = parseFlags([rawArgs]);
  }

  let weekFilter = flags.week;
  let dayFilter = flags.day;

  const pos = flags._.join(' ').trim();
  if (pos) {
    if (pos.toLowerCase().startsWith('week')) {
      const parts = pos.split(/\s+/);
      if (parts[1]) weekFilter = parts[1];
    } else if (pos.toLowerCase().startsWith('day')) {
      dayFilter = pos;
      const m = pos.match(/day\s*(\d+)/i);
      if (m && !weekFilter) {
        weekFilter = Math.ceil(parseInt(m[1], 10) / 7);
      }
    } else if (['1', '2', '3', '4', '5'].includes(pos)) {
      weekFilter = pos;
    } else {
      dayFilter = pos;
    }
  }

  if (dayFilter && !weekFilter) {
    const m = dayFilter.match(/day\s*(\d+)/i);
    if (m) {
      weekFilter = Math.ceil(parseInt(m[1], 10) / 7);
    }
  }

  const { filePath, month } = getActiveTaskFile(weekFilter);
  const content = fs.readFileSync(filePath, 'utf8');
  const { tasks } = parseTasks(content);

  console.log(`\n${BOLD}${CYAN}✦ BramandScript Task Dashboard (Month: ${month})${RESET}`);
  console.log(`${DIM}File: ${filePath}${RESET}\n`);

  let currentDay = '';
  let completed = 0;
  let matchedTasks = 0;

  tasks.forEach(t => {
    if (t.done) completed++;

    if (dayFilter && !t.day.toLowerCase().includes(dayFilter.toLowerCase())) {
      return;
    }

    matchedTasks++;
    if (t.day !== currentDay) {
      currentDay = t.day;
      console.log(`\n${BOLD}${currentDay}${RESET}`);
      console.log(`${DIM}${'─'.repeat(45)}${RESET}`);
    }

    const statusBadge = t.done 
      ? `${GREEN}[OK COMPLETE]${RESET}` 
      : `${YELLOW}[· PENDING ]${RESET}`;
    const num = String(t.index).padStart(2, '0');
    console.log(`  ${DIM}${num}.${RESET} ${statusBadge} ${t.done ? DIM + t.text + RESET : t.text}`);
  });

  const total = tasks.length;
  const pct = total > 0 ? Math.round((completed / total) * 100) : 0;
  console.log(`\n${DIM}${'═'.repeat(45)}${RESET}`);
  if (dayFilter) {
    console.log(`${BOLD}Showing:${RESET} ${matchedTasks} task(s) matching "${dayFilter}"`);
  }
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
  console.log(`${BOLD}Status:${RESET}      ${targetTask.done ? GREEN + '[OK COMPLETE]' + RESET : YELLOW + '[· PENDING ]' + RESET}`);
  console.log(`${BOLD}Specification:${RESET} ${targetTask.text}\n`);
}

function parseFlags(rawArgs) {
  const flags = {
    _: [],
    day: null,
    desc: null,
    includes: null,
    week: null,
  };

  for (let i = 0; i < rawArgs.length; i++) {
    const arg = rawArgs[i];
    if (arg.startsWith('--day=')) {
      flags.day = arg.slice(6);
    } else if (arg === '--day' && i + 1 < rawArgs.length) {
      flags.day = rawArgs[++i];
    } else if (arg.startsWith('--week=')) {
      flags.week = arg.slice(7);
    } else if (arg === '--week' && i + 1 < rawArgs.length) {
      flags.week = rawArgs[++i];
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

function cmdAdd(title, dayArg, descArg, includesArg, weekArg) {
  if (!title || !title.trim()) {
    console.error(`${RED}[ERROR] Task title cannot be empty!${RESET}`);
    console.error(`${YELLOW}Usage: bmd-task add "Task title" --desc="Detailed description" [--includes="..."] [--day=N] [--week=N]${RESET}`);
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
    console.error(`  ${CYAN}--day=N${RESET}            : Target day (e.g., --day=8).\n`);
    console.error(`${BOLD}Example:${RESET}`);
    console.error(`  bmd-task add "Standard HTTP Router" --day=8 \\`);
    console.error(`    --desc="Build high-performance multiplexer with route matching and middleware support" \\`);
    console.error(`    --includes="Route tree, pattern matching, handler context"\n`);
    process.exit(1);
  }

  let targetWeek = weekArg;
  if (!targetWeek && dayArg) {
    const d = parseInt(dayArg, 10);
    if (!isNaN(d) && d > 0) {
      targetWeek = Math.ceil(d / 7);
    }
  }

  const { filePath } = getActiveTaskFile(targetWeek, true);
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
  console.log(`${BOLD}${GREEN}[OK]${RESET} Task added to ${targetDay} in ${path.basename(filePath)}:`);
  console.log(`  ${BOLD}Title:${RESET}         ${title.trim()}`);
  console.log(`  ${BOLD}Description:${RESET}   ${cleanDesc}`);
  if (includesArg && includesArg.trim()) {
    console.log(`  ${BOLD}Includes:${RESET}      ${includesArg.trim()}`);
  }
}

function cmdInitWeek(weekNum, mmyy) {
  const dateDir = path.join(process.cwd(), 'date');
  if (!fs.existsSync(dateDir)) {
    fs.mkdirSync(dateDir, { recursive: true });
  }

  const months = fs.readdirSync(dateDir).filter(f => fs.statSync(path.join(dateDir, f)).isDirectory());
  months.sort().reverse();
  let latestMonth = mmyy || (months.length > 0 ? months[0] : null);

  if (!latestMonth) {
    const now = new Date();
    const mm = String(now.getMonth() + 1).padStart(2, '0');
    const yy = String(now.getFullYear()).slice(-2);
    latestMonth = `${mm}${yy}`;
  }

  const targetDir = path.join(dateDir, latestMonth);
  if (!fs.existsSync(targetDir)) {
    fs.mkdirSync(targetDir, { recursive: true });
  }

  const weekNames = ['First', 'Second', 'Third', 'Fourth', 'Fifth'];
  const num = parseInt(weekNum, 10) || 2;
  if (num < 1 || num > 5) {
    console.error(`${RED}[ERROR] Week number must be between 1 and 5${RESET}`);
    process.exit(1);
  }

  const weekName = weekNames[num - 1];
  const targetFile = path.join(targetDir, `${weekName}_week_task.md`);

  if (fs.existsSync(targetFile)) {
    console.log(`${YELLOW}[WARN] Task file already exists: ${targetFile}${RESET}`);
    return;
  }

  const startDay = (num - 1) * 7 + 1;
  const endDay = num * 7;
  let daysContent = '';
  for (let d = startDay; d <= endDay; d++) {
    daysContent += `## Day ${d}\n\n`;
  }

  const template = `# BramandScript — Month ${latestMonth}: ${weekName} Week Task (Days ${startDay} to ${endDay})\n\n${daysContent}`;
  fs.writeFileSync(targetFile, template, 'utf8');
  console.log(`${BOLD}${GREEN}[OK]${RESET} Initialized ${weekName} Week task file (Days ${startDay} to ${endDay}):\n  ${DIM}${targetFile}${RESET}`);
}

function cmdInit(arg1, arg2) {
  // If arg1 is a number (1-5) or week name, initialize that week
  const weekNum = parseInt(arg1, 10);
  if (!isNaN(weekNum) && weekNum >= 1 && weekNum <= 5) {
    cmdInitWeek(weekNum, arg2);
    return;
  }
  const weekNames = ['first', 'second', 'third', 'fourth', 'fifth'];
  if (arg1 && weekNames.includes(arg1.toLowerCase())) {
    cmdInitWeek(weekNames.indexOf(arg1.toLowerCase()) + 1, arg2);
    return;
  }

  const now = new Date();
  const defaultMM = String(now.getMonth() + 1).padStart(2, '0');
  const defaultYY = String(now.getFullYear()).slice(-2);
  const targetMMYY = arg1 || `${defaultMM}${defaultYY}`;

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
    cmdList(args.slice(1));
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
    cmdAdd(titleArg, flags.day, flags.desc, flags.includes, flags.week);
    break;

  case 'init-week':
  case 'new-week':
  case 'add-week':
    cmdInitWeek(args[1], args[2]);
    break;

  case 'init':
    cmdInit(args[1], args[2]);
    break;

  case 'help':
  case '--help':
  case '-h':
    console.log(`
${BOLD}✦ BramandScript Task & Todo CLI Tool${RESET}

${BOLD}Commands:${RESET}
  bmd-task list [day | week]                           List tasks (e.g., 'bmd-task list', 'bmd-task list Day 8', 'bmd-task list 2')
  bmd-task view <id | text>                            View task full description & details
  bmd-task check <id | text>                           Mark task as complete [x]
  bmd-task uncheck <id | text>                         Mark task as pending [ ]
  bmd-task edit <id | text> "New Task Text"            Update existing task text
  bmd-task add "Title" --desc="..." [--day=N]          Add new task (${BOLD}detailed --desc is COMPULSORY${RESET})
  bmd-task init-week <1-5>                             Initialize weekly task file (e.g., 'bmd-task init-week 2' for Days 8-14)
  bmd-task init [MMYY | week_number]                   Initialize monthly tracking folder or week file

${BOLD}Flags for 'add' command:${RESET}
  ${CYAN}--desc="..."${RESET}       ${BOLD}[COMPULSORY]${RESET} Detailed description of what must be implemented.
  ${CYAN}--includes="..."${RESET}   ${DIM}[RECOMMENDED]${RESET} Deliverables, components, or test scopes.
  ${CYAN}--day=N${RESET}            Target day number (e.g., --day=8 automatically targets Week 2).
  ${CYAN}--week=N${RESET}           Target week number (1 to 5).
`);
    break;

  default:
    console.error(`${RED}Unknown command: ${command}${RESET}. Run 'bmd-task help' for usage.`);
    process.exit(1);
}
