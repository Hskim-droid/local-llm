#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
DEST="${HOME}/.local/bin"
mkdir -p "$DEST"
chmod +x "$ROOT/report" "$ROOT/reportctl.py"
ln -sfn "$ROOT/report" "$DEST/report"
case ":$PATH:" in
  *":$DEST:"*) ;;
  *) echo "Add this to ~/.zshrc then open a new terminal:"; echo "  export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac
echo "Installed. New terminal:"
echo "  report"
echo "then drop a video, PPTX, and PDF, press Enter."
