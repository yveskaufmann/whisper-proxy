FROM debian:trixie-slim AS script-generator

RUN <<EOF
cat << 'INNER_EOF' > /setup-rocm-repo.sh
#!/bin/bash
set -e

# Install dependencies for adding the ROCm and AMDGPU repositories
apt-get update && apt-get install -y --no-install-recommends ca-certificates curl gnupg libnuma-dev

# Download keys and add them to the keyring
mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.radeon.com/rocm/rocm.gpg.key | gpg --dearmor | tee /etc/apt/keyrings/rocm.gpg > /dev/null

# Add source file for ROCm and AMDGPU repositories
printf "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/${ROCM_VERSION}/ noble main\n" | tee /etc/apt/sources.list.d/rocm.list
printf "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/graphics/${AMDGPU_VERSION}/ubuntu noble main\n" | tee /etc/apt/sources.list.d/amdgpu.list

# Ensure that the ROCm and AMDGPU packages have a higher priority than other packages
printf "Package: *\nPin: release o=repo.radeon.com\nPin-Priority: 600\n" > /etc/apt/preferences.d/rocm-pin-600

apt-get update
INNER_EOF
chmod +x /setup-rocm-repo.sh
EOF


FROM golang:1.26.4-trixie AS builder

WORKDIR /app

# RCOM and AMDGPU versions
ARG ROCM_VERSION=7.2.4
ARG AMDGPU_VERSION=7.2.4

# Install ROCm and AMDGPU repositories and dependencies
COPY --from=script-generator /setup-rocm-repo.sh /usr/local/bin/
RUN setup-rocm-repo.sh && apt-get install -y --no-install-recommends --allow-downgrades \
    build-essential cmake \
    python3-setuptools python3-wheel \
    rocm-hip-sdk rocm-ml-sdk \
    && apt-get clean  \
    && rm -rf /var/lib/apt/lists/*

COPY third_party/whisper.cpp /app/third_party/whisper.cpp

RUN cd /app/third_party/whisper.cpp \
    && cmake -B build -DGGML_HIP=1 -DAMDGPU_TARGETS="gfx1201" -DCMAKE_BUILD_TYPE=Release \
    && cmake --build build --config Release --parallel $(nproc)

COPY go.mod go.sum ./
RUN go mod download && go mod verify

ENV C_INCLUDE_PATH=/app/third_party/whisper.cpp/include:/app/third_party/whisper.cpp/ggml/include
ENV LIBRARY_PATH=/app/third_party/whisper.cpp/build/bin
ENV CGO_ENABLED=1

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux make build 

FROM debian:trixie-slim

WORKDIR /app

ARG ROCM_VERSION=7.2.4
ARG AMDGPU_VERSION=7.2.4

# Install ROCm and AMDGPU repositories and dependencies
COPY --from=script-generator /setup-rocm-repo.sh /usr/local/bin/
RUN setup-rocm-repo.sh && apt-get dist-upgrade -y && apt-get install -y --no-install-recommends --allow-downgrades \
    ffmpeg rocm-hip-runtime rocm-hip-libraries \
    && apt-get clean  \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/third_party/whisper.cpp/build/bin/ /usr/local/lib/
COPY --from=builder /app/bin/whisper-proxy /app/whisper-proxy

ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

EXPOSE 8080

ENTRYPOINT ["/app/whisper-proxy"]