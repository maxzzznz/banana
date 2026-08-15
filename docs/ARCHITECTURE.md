# Architecture and operational notes

## Scope

The first release targets one model only: Gemini Nano Banana 2 Lite (`gemini-3.1-flash-lite-image`). Its output size is fixed at 1K, so the CLI deliberately does not expose a size flag.

The public command surface is intentionally small:

- one text prompt;
- an optional aspect ratio;
- zero or one reference image;
- one required output path.

## Dependencies

The implementation uses only the Go standard library and calls the Gemini Interactions REST API directly. This produces a portable executable without a Python runtime, package manager, or SDK dependency.

## API flow

1. Validate all local inputs before making a network call.
2. Load the API key from `GEMINI_API_KEY` or the saved user configuration.
3. Base64-encode the optional image and send it with the prompt.
4. Request PNG output and the selected aspect ratio.
5. Retry only transient failures.
6. Decode the returned image and write it to the requested path.
7. Emit a single JSON result.

## Future extensions

If additional Gemini image models are introduced, add an explicit model registry before exposing `--model` and `--size`. Each registry entry should define supported ratios, allowed output sizes, and reference-image limits. That prevents the CLI from accepting a valid-looking combination that a chosen model cannot serve.

Avoid changing the JSON result fields incompatibly; agents benefit from a stable machine-readable contract.
