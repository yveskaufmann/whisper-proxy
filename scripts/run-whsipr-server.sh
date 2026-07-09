#!/usr/bin/env bash



LOCAL_PORT=18080
LOCAL_MODELS="$HOME/.config/whisper-server/models"
MODEL_TO_USE="/models/ggml-large-v3-turbo.bin"
IMAGE="ghcr.io/ggml-org/whisper.cpp:main-vulkan"

mkdir -p "$LOCAL_MODELS" || true > /dev/null 

WHISPER_ARGS=(
    # Host of the server 
    "--host" "0.0.0.0"
    # Port to bind the server.
    "--port" "$LOCAL_PORT"
    # Automatic language detection 
    "-l" "auto"
    # Model to use 
    "-m" "${MODEL_TO_USE}"
		# Enable Voice Activity detection
    #"--vad"
)

docker run -it --rm -p "$LOCAL_PORT:$LOCAL_PORT" \
  --device=/dev/kfd --device=/dev/dri \
  --volume "$LOCAL_MODELS:/models" \
  --entrypoint "/app/build/bin/whisper-server" \
  $IMAGE "${WHISPER_ARGS[@]}"
