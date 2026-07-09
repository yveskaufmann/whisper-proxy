# whisper-proxy

A small proxy that puts an OpenAI-compatible `/audio/transcriptions` endpoint in front of [whisper.cpp](https://github.com/ggml-org/whisper.cpp), so it can be used as a drop-in Speech-to-Text backend for [open-whispr](https://github.com/) with GPU acceleration on AMD hardware.

## Motivation

open-whispr only knows how to leverage the GPU when it's talking to an Nvidia (CUDA) backend. There is no way to point it at a whisper.cpp server accelerated via ROCm/Vulkan on an AMD card (e.g. a Radeon GPU).

open-whispr does let you configure which server it talks to, though. This project exploits that: it exposes an OpenAI-style API (the same shape open-whispr expects), and internally forwards the audio to a whisper.cpp instance built with ROCm/HIP support so transcription actually runs on the Radeon GPU. Point open-whispr at this proxy instead of OpenAI, and you get local, GPU-accelerated transcription without an Nvidia card.

## How it works

- `whisper-proxy` exposes:
  - `POST /audio/transcriptions` — OpenAI-compatible transcription endpoint
  - `GET /health` — health check
- It can transcribe audio in two ways:
  - **Embedded**: loads a GGML model directly via the whisper.cpp Go bindings (`-m/--model`) and runs inference in-process.
  - **Remote**: forwards requests to a separate `whisper.cpp` example server (`--whisper-url`), e.g. the ROCm/Vulkan-accelerated `whisper-server` container.

## Prerequisites

- Go 1.24+
- Git (with submodule support)
- A whisper GGML model (e.g. `ggml-large-v3-turbo.bin`), downloadable via whisper.cpp's `models/download-ggml-model.sh`
- For GPU acceleration: an AMD GPU with ROCm/Vulkan drivers, or Docker if you'd rather not install ROCm on the host

This repo includes `whisper.cpp` as a git submodule under `third_party/whisper.cpp`.

## Building locally

```bash
# Clone with submodules (or `git submodule update --init --recursive` afterwards)
git clone --recurse-submodules <this-repo-url>
cd open-whsipr-proxy

# Build the whisper.cpp shared library (adjust flags for your GPU backend, e.g. -DGGML_HIP=1 for ROCm)
cd third_party/whisper.cpp
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release --parallel $(nproc)
cd ../..

# Build the proxy binary (Makefile wires up C_INCLUDE_PATH/LIBRARY_PATH for the whisper.cpp bindings)
make build
```

This produces `bin/whisper-proxy`.

Other useful targets:

```bash
make run    # go run the proxy (ARGS="--port 8080 -m /path/to/model.bin")
make tidy   # go mod tidy
make fmt    # gofmt + goimports
make clean  # remove built binaries
```

## Running

Embedded mode (whisper.cpp runs in-process, needs the shared library built above on `LD_LIBRARY_PATH`):

```bash
./bin/whisper-proxy --port 8080 -m /path/to/ggml-large-v3-turbo.bin
```

Proxy mode (forwards to an existing whisper.cpp server):

```bash
./bin/whisper-proxy --port 8080 --whisper-url http://localhost:18080
```

Then point open-whispr's custom API endpoint at `http://localhost:8080`.

## Docker / GPU-accelerated build

The included `Dockerfile` builds whisper.cpp with ROCm/HIP support (`-DGGML_HIP=1`) targeting Radeon GPUs (default `AMDGPU_TARGETS=gfx1201` — adjust for your GPU's architecture) and bundles it with the proxy binary.

```bash
docker compose up --build
```

`compose.yaml` mounts `$HOME/.config/whisper-server/models` into the container for models, and passes through `/dev/dri` and `/dev/kfd` so the container can access the AMD GPU. By default it runs `whisper-proxy` in embedded mode; there's also an (initially scaled-to-0) `whisper-backend` service using the upstream `whisper.cpp` Vulkan image if you'd rather run the proxy in forwarding mode against a separate server.

## Configuration flags

| Flag            | Default                  | Description                                                          |
| --------------- | ------------------------ | -------------------------------------------------------------------- |
| `--port`        | `8080`                   | Port the proxy listens on                                            |
| `--whisper-url` | `http://localhost:18080` | URL of a whisper.cpp backend server (used when `--model` is not set) |
| `-m`, `--model` | _(none)_                 | Path to a GGML model to load and run in-process                      |
| `--version`     | `false`                  | Print version info and exit                                          |
