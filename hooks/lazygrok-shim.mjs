#!/usr/bin/env node
// lazygrok-shim.mjs — translates Grok hook events to Codex format for lazygrok hooks.
//
// Grok sends: {"event":"UserPromptSubmit","prompt":"...","session_id":"...","workspace":"..."}
// Codex expects: {"hook_event_name":"UserPromptSubmit","prompt":"...","session_id":"...",
//                 "turn_id":"...","transcript_path":null,"cwd":"...","model":"...",
//                 "permission_mode":"default","source":"startup"}
//
// Usage: node lazygrok-shim.mjs <component-name> <hook-event>
// Example: node lazygrok-shim.mjs ultrawork user-prompt-submit

import { spawn } from "node:child_process";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { env } from "node:process";
import { readFileSync } from "node:fs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const PLUGIN_ROOT = env.GROK_PLUGIN_ROOT || resolve(__dirname, "..");

const componentName = process.argv[2];
const hookEvent = process.argv[3];

if (!componentName || !hookEvent) {
  process.stderr.write("Usage: lazygrok-shim.mjs <component> <hook-event>\n");
  process.exit(1);
}

const cliPath = resolve(PLUGIN_ROOT, "vendor/lazygrok-hooks", componentName, "dist/cli.js");

// Read all of stdin
const chunks = [];
process.stdin.on("data", (chunk) => chunks.push(chunk));
process.stdin.on("end", () => {
  const raw = Buffer.concat(chunks).toString("utf8").trim();
  if (!raw) {
    process.exit(0);
  }

  let input;
  try {
    input = JSON.parse(raw);
  } catch {
    process.stdout.write(raw);
    process.exit(0);
  }

  // Map Grok event field names to Codex field names
  const grokEvent = input.event || input.hookEventName || input.hook_event_name || "";
  const sessionId = input.session_id || input.sessionId || input.sessionID || "";
  const workspace = input.workspace || input.workspaceRoot || input.cwd || process.cwd();
  const prompt = input.prompt || input.userPrompt || input.user_prompt || input.message || "";
  const toolName = input.tool || input.toolName || input.tool_name || "";
  const toolInput = input.tool_input || input.toolInput || input.arguments || input.input || {};
  const toolUseId = input.tool_use_id || input.toolUseId || input.toolCallId || "";
  const stopHookActive = input.stop_hook_active || input.stopHookActive || false;
  const subagentId = input.subagent_id || input.subagentId || "";

  // Build the Codex-format event
  const codexEvent = {
    hook_event_name: grokEvent,
    session_id: sessionId,
    turn_id: sessionId, // Grok doesn't have turn_id, use session_id
    transcript_path: null, // Grok doesn't expose transcript path
    cwd: workspace,
    model: input.model || "grok-build",
    permission_mode: input.permission_mode || "default",
  };

  // Add event-specific fields
  if (grokEvent === "UserPromptSubmit") {
    codexEvent.prompt = prompt;
  } else if (grokEvent === "PreToolUse" || grokEvent === "PostToolUse") {
    codexEvent.tool_name = toolName;
    codexEvent.tool_use_id = toolUseId;

    // For PostToolUse on write/edit tools, the lazycodex hooks expect
    // the file content in tool_input.content. Grok doesn't include it,
    // so we read the file from disk and inject it.
    let enrichedToolInput = { ...toolInput };
    const filePath = enrichedToolInput.file_path || enrichedToolInput.filePath || enrichedToolInput.path || "";
    const lowerTool = toolName.toLowerCase();
    if (grokEvent === "PostToolUse" && filePath && !enrichedToolInput.content &&
        (lowerTool === "write" || lowerTool === "edit" || lowerTool === "search_replace" ||
         lowerTool === "strreplace" || lowerTool === "apply_patch" || lowerTool === "multiedit")) {
      try {
        const absPath = resolve(workspace, filePath);
        enrichedToolInput.content = readFileSync(absPath, "utf8");
      } catch {
        // File may not exist yet or may be outside workspace — skip enrichment
      }
    }
    codexEvent.tool_input = enrichedToolInput;
  } else if (grokEvent === "Stop") {
    codexEvent.stop_hook_active = stopHookActive;
  } else if (grokEvent === "SubagentStop") {
    codexEvent.subagent_id = subagentId;
  } else if (grokEvent === "SessionStart") {
    codexEvent.source = input.source || "startup";
  } else if (grokEvent === "PostCompact") {
    codexEvent.trigger = input.trigger || "auto";
  }

  // Spawn the actual hook component with the translated input
  // Set CODEX_HOME if not already set — the lsp hook needs it to find the daemon socket
  const childEnv = { ...env };
  if (!childEnv.CODEX_HOME) {
    childEnv.CODEX_HOME = env.HOME ? `${env.HOME}/.codex` : "/tmp/codex-home";
  }
  const child = spawn("node", [cliPath, "hook", hookEvent], {
    stdio: ["pipe", "inherit", "inherit"],
    env: { ...childEnv, PLUGIN_ROOT: resolve(PLUGIN_ROOT, "vendor/lazygrok-hooks", componentName) },
  });

  child.stdin.write(JSON.stringify(codexEvent));
  child.stdin.end();

  child.on("exit", (code) => {
    process.exit(code || 0);
  });
});
