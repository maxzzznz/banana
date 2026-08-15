# nanobanana

Small, dependency-free Go command-line tool for generating an image with Gemini Nano Banana 2 Lite (`gemini-3.1-flash-lite-image`). It uses the Gemini Interactions API directly, so the compiled executable has no runtime dependency.

## Requirements

- Go 1.22+ to build it.
- A Gemini API key. You can use `GEMINI_API_KEY`, or save it once with the configuration command below.

## Build

```powershell
$env:GEMINI_API_KEY = "your-key"
go build -o bin\\nanobanana.exe .
```

For a different machine or operating system, cross-compile a single executable. Example for macOS on Apple Silicon:

```powershell
$env:GOOS = "darwin"; $env:GOARCH = "arm64"
go build -o nanobanana .
```

## Use

### Configure once (optional)

```powershell
.\nanobanana.exe configure
```

The command asks for a Gemini API key once. It saves it in the current user's application configuration directory (`AppData` on Windows), never in the project folder. This avoids accidentally committing or copying the secret with the executable. The configuration file is created with owner-only permissions where the operating system supports them.

For automated agents and CI, `GEMINI_API_KEY` remains the preferred mechanism and takes precedence over the saved key.

### Generate

```powershell
.\nanobanana.exe --prompt "A small astronaut walking through a sunflower field" --ratio 16:9 --output outputs\astronaut.png
```

For long or multiline prompts, use a file:

```powershell
.\nanobanana.exe --prompt-file brief.txt --ratio 16:9 --output outputs\astronaut.png
```

Or pipe the prompt through standard input. This is useful when an AI agent produces the prompt itself:

```powershell
"A small astronaut walking through a sunflower field" | .\nanobanana.exe --ratio 16:9 --output outputs\astronaut.png
```

To use one optional reference image:

```powershell
.\nanobanana.exe --prompt "Turn this into a watercolor illustration" --input reference.jpg --ratio 4:5 --output outputs\watercolor.png
```

The command emits exactly one JSON object to standard output. Errors are also JSON and use a non-zero exit code, making the tool suitable for an AI agent.

## Interface

- exactly one prompt source: `--prompt`, `--prompt-file`, or piped standard input
- `--output` required PNG destination
- `--ratio` optional: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, or `21:9`
- `--input` optional single reference image (maximum 20 MB)
- `--timeout` optional total deadline; default: 90 seconds

Nano Banana 2 Lite only produces 1K images, so this initial CLI intentionally has no size option. The implementation isolates the model and response format so a future `--model` and `--size` extension can be added without changing the basic invocation.

The client retries transient network failures, rate limits, and server errors up to four times with backoff. It never prints the API key.
