#!/bin/bash

OS=$(uname -s)
if [ "$OS" != "Linux" ]; then
    echo "本脚本仅支持 Linux/Android 平台。"
    exit 1
fi

ARCH=$(uname -m)
case "$ARCH" in
    aarch64|arm64) ARCH="arm64" ;;
    armv7*|armv8*) ARCH="arm" ;;
    x86_64)        ARCH="amd64" ;;
    i386|i686)     ARCH="386" ;;
    *)             echo "不支持的架构: $ARCH" && exit 1 ;;
esac

BINARY="BPB-Wizard"
ARCHIVE="${BINARY}-${OS}-${ARCH}.tar.gz"
LATEST_VERSION=$(curl -fsSL https://raw.githubusercontent.com/zatursure/BPB-Wizard-Chinese/main/VERSION)

if [ -x "./${BINARY}" ]; then
    INSTALLED_VERSION=$("./${BINARY}" --version)
    echo "已安装版本: $INSTALLED_VERSION"
    echo "最新版本: ${LATEST_VERSION}"

    if [ "${INSTALLED_VERSION}" = "${LATEST_VERSION}" ]; then
        echo "向导已是最新版本，正在运行..."
        exec ./"${BINARY}"
    else
        echo "正在更新到版本 ${LATEST_VERSION}..."
    fi
else
    echo "未检测到向导，正在安装版本 ${LATEST_VERSION}..."
fi

echo "正在下载 ${ARCHIVE}..."
curl -L -# -o "${ARCHIVE}" "https://github.com/zatursure/BPB-Wizard-Chinese/releases/latest/download/${ARCHIVE}" && \
tar xzf "${ARCHIVE}" && \
chmod +x "./${BINARY}" && \
exec ./"${BINARY}"