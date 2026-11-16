#!/bin/sh
# Installe la CLI Mercy Pay : curl -fsSL https://raw.githubusercontent.com/mercy-pay/cli/main/install.sh | sh
set -eu

REPO="mercy-pay/cli"
INSTALL_DIR="${MERCYPAY_INSTALL_DIR:-/usr/local/bin}"

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) echo "Système non pris en charge : téléchargez l'archive sur https://github.com/$REPO/releases" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) echo "Processeur non pris en charge : $(uname -m)" >&2; exit 1 ;;
esac

version="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p' | head -n 1)"
if [ -z "$version" ]; then
  echo "Impossible de trouver la dernière version." >&2
  exit 1
fi

archive="mercypay_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/v${version}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Téléchargement de mercypay ${version} (${os}/${arch})…"
curl -fsSL "$base/$archive" -o "$tmp/$archive"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

expected="$(grep " $archive\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)"
else
  actual="$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)"
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
  echo "Somme de contrôle différente : installation annulée." >&2
  exit 1
fi

tar -xzf "$tmp/$archive" -C "$tmp" mercypay
if [ -w "$INSTALL_DIR" ]; then
  mv "$tmp/mercypay" "$INSTALL_DIR/mercypay"
else
  sudo mv "$tmp/mercypay" "$INSTALL_DIR/mercypay"
fi

echo "Installé dans $INSTALL_DIR/mercypay. Commencez par : mercypay login"
