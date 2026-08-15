# Using banana from an AI agent

`banana` is designed as a small command boundary: an agent supplies the creative inputs and receives a JSON result, while the executable reads its own Gemini API credential.

## Contract

The command exits with code `0` on success and prints one JSON object to standard output:

```json
{"ok":true,"output":"outputs/example.jpg","format":"jpeg","model":"gemini-3.1-flash-lite-image","aspect_ratio":"16:9","reference_image":false}
```

On failure it exits non-zero and still prints one JSON object:

```json
{"ok":false,"error":"human-readable error"}
```

Do not parse human prose. Read `ok`, then use `output` when it is `true` or `error` when it is `false`.

## Recommended invocation

Use standard input for agent-generated prompts. It preserves newlines and avoids shell-quoting problems.

```text
<prompt text> | banana --ratio 16:9 --output generated.jpg
```

For a reference-driven request, add one local image path:

```text
<prompt text> | banana --input source.png --ratio 4:5 --output generated.jpg
```

The tool accepts exactly one prompt source: `--prompt`, `--prompt-file`, or standard input. It accepts zero or one reference image.

JPEG is the default output format. Pass `--format png` with a `.png` output path when lossless PNG output is needed. The CLI rejects mismatched output extensions so agents do not receive incorrectly named files.

## Credentials

An agent should not be given an API key in its prompt, command arguments, or standard input. Configure the executable once in the user environment with:

```text
banana configure
```

For unattended deployment, the hosting environment may provide `GEMINI_API_KEY`. The executable does not print the key.

This protects the key from normal tool inputs and logs. It is not a security boundary against an agent that can freely read every file or execute arbitrary commands as the same operating-system user.

Use `banana status` to confirm whether a usable key is available. The JSON result never includes the key itself. Use `banana logout` to remove only the locally saved key; it does not alter a `GEMINI_API_KEY` supplied by the runtime environment.

## Retry behavior

The tool retries transient network errors, rate limits, and server-side failures up to four times with exponential backoff. Invalid requests and authentication failures return immediately.
