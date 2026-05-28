# tuna

🐟 Dead simple prompt tuner.

`tuna` helps compare how multiple LLMs respond to the same prompts. It creates an assistant workspace, generates an execution plan, runs queries against one or more models, and stores outputs in a predictable folder structure.

## What it does

- Create an assistant workspace with folders for system prompts, input prompts, and outputs.
- Build a plan for one or more models.
- Execute the same prompt set across multiple models.
- Save model responses for side-by-side review.
- Route models through configurable providers and aliases via `.tuna.toml`.
- Optionally control provider-specific thinking behavior per model through `chat_template_kwargs.enable_thinking`.

## Install

```bash
go install go.octolab.org/toolset/tuna@latest
```

## Quick start

### 1. Create an assistant

```bash
tuna init my-assistant
```

This creates a workspace like:

```text
my-assistant/
├── System prompt/
├── Input/
└── Output/
```

Add your system prompt content under `System prompt/` and your test prompts under `Input/`.

### 2. Create a plan

```bash
tuna plan my-assistant --models qwen3,deepseek
```

You can also tune generation settings when the plan is created:

```bash
tuna plan my-assistant \
  --models qwen3,deepseek \
  --temperature 0.2 \
  --max-tokens 8000
```

`temperature` controls how deterministic or creative the responses are. `max-tokens` limits the maximum response length for each model run.

### 3. Execute the plan

```bash
tuna exec my-assistant
```

Generated responses are written under the assistant's `Output/` directory.

## Configuration

`tuna` reads provider and alias configuration from `.tuna.toml`.

Example:

```toml
default_provider = "openrouter"

[aliases]
qwen3 = "qwen/qwen3-32b"
deepseek = "deepseek/deepseek-r1"
gpt4o = "openai/gpt-4o"

[[providers]]
name = "openrouter"
base_url = "https://openrouter.ai/api/v1"
api_token_env = "OPENROUTER_API_KEY"
rate_limit = "10rpm"
models = [
  "qwen/qwen3-32b",
  "deepseek/deepseek-r1",
  "openai/gpt-4o",
]

[[providers.model_options]]
name = "qwen/qwen3-32b"
enable_thinking = false

[[providers.model_options]]
name = "deepseek/deepseek-r1"
enable_thinking = true
```

### Provider fields

- `default_provider`: provider used when a model is not explicitly mapped.
- `aliases`: short names mapped to full model identifiers.
- `providers[].name`: logical provider name.
- `providers[].base_url`: OpenAI-compatible API base URL.
- `providers[].api_token` or `providers[].api_token_env`: authentication source.
- `providers[].rate_limit`: optional rate limit such as `10rpm`, `5rps`, or `100rph`.
- `providers[].models`: models served by that provider.
- `providers[].model_options`: optional per-model request settings.

### Per-model thinking mode

For compatible gateways, `tuna` can send model-specific thinking controls in the outgoing chat-completions request:

```json
{
  "chat_template_kwargs": {
    "enable_thinking": false
  }
}
```

Use one `[[providers.model_options]]` block per model you want to override.

Rules:

- `name` must match a model listed in the same provider's `models` array.
- `enable_thinking = false` disables thinking for that model.
- `enable_thinking = true` enables thinking for that model.
- If no `model_options` block exists for a model, `tuna` leaves `chat_template_kwargs` out of the request.

## Command summary

### `tuna init`

Create a new assistant workspace.

```bash
tuna init my-assistant
```

### `tuna plan`

Generate a plan for one or more models.

```bash
tuna plan my-assistant --models qwen3,deepseek
```

Useful flags:

- `--models`: comma-separated model aliases or full model names.
- `--temperature`: generation temperature, default `0.7`.
- `--max-tokens`: maximum response tokens, default `4096`.

### `tuna exec`

Execute the current plan and save responses.

```bash
tuna exec my-assistant
```

Useful flags depend on your current version, but the main workflow is always: create an assistant, generate a plan, then execute it.

## Typical workflow

```bash
tuna init okr-review

# Add prompt files under:
#   okr-review/System prompt/
#   okr-review/Input/

tuna plan okr-review --models qwen3,deepseek --temperature 0.1 --max-tokens 6000

:wqtuna exec okr-review
```

## Notes

- Model aliases are resolved before execution.
- Provider routing is based on the configured `models` list and default provider.
- Responses from all runs are stored on disk for later comparison.
- Thinking mode is a provider-compatible extension and only affects models with explicit `model_options` configuration.
