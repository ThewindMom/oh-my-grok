# Configuration Reference

oh-my-grok uses typed JSONC configuration with documented precedence.

## Precedence (highest first)

1. **Environment overrides** (`OMG_*` variables)
2. **Workspace config**: `.omg/config.jsonc` in the workspace root
3. **User config**: `~/.grok/oh-my-grok/config.jsonc` (or `$GROK_HOME/oh-my-grok/config.jsonc`)
4. **Built-in defaults**

## File format

Configuration files use JSONC (JSON with comments and trailing commas):

```jsonc
{
  // Hashline enforcement mode
  "hashlineMode": "prefer",  // off | prefer | strict

  // Continuation
  "continuationEnabled": true,
  "maxContinuations": 25,
  "cooldownSeconds": 10,
  "repeatedStateThreshold": 3,

  // Loops
  "ralphEnabled": true,
  "ultraworkEnabled": true,

  // Enforcement
  "todoEnforcement": true,
  "boulderEnforcement": true,
  "planEnforcement": true,
  "skillGateEnabled": true,
  "intentGateEnabled": true,

  // LSP
  "lspEnabled": true,
  "lspStopEnforcement": false,

  // Policies
  "commentPolicy": "allow",  // allow | warn | deny
  "projectRuleInjection": true,

  // Context limits (bytes)
  "context": {
    "sectionBytes": 4096,
    "maxBytes": 32768
  },

  // Orchestration
  "subagentConcurrency": 4,
  "worktreeIsolation": false,

  // State and logging
  "stateRetention": "7d",
  "logLevel": "info",  // error | warn | info | debug
  "logPath": "",

  // Disabled components
  "disabledHooks": [],
  "disabledAgents": [],
  "disabledCommands": [],
  "disabledSkills": []
}
```

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OMG_HASHLINE` | Hashline mode (`off`, `prefer`, `strict`) | `prefer` |
| `OMG_INTENT_GATE` | Enable intent gate | `true` |
| `OMG_LSP_ENFORCE` | Enable LSP stop enforcement | `false` |
| `OMG_MAX_CONTINUATIONS` | Max continuation iterations | `25` |
| `OMG_COOLDOWN_SECONDS` | Continuation cooldown | `10` |
| `OMG_RALPH` | Enable Ralph loop | `true` |
| `OMG_ULTRAWORK` | Enable Ultrawork | `true` |
| `OMG_CONTINUATION` | Enable continuation | `true` |

## Unknown keys

Unknown configuration keys produce diagnostics rather than silently changing behavior. Check the doctor output or diagnostic logs for unknown key warnings.

## Invalid values

Invalid values fail validation with a precise message. Use `omg-hook doctor` to check configuration validity.
