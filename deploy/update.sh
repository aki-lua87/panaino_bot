#!/usr/bin/env bash
# Ubuntu 16.04 / Bash 4.3 compatible. Run as the owner of the Bot directory.
set -euo pipefail
umask 077

usage() {
    echo 'Usage: bash update.sh [latest|build-COMMIT_SHA] [--stage]'
    echo '       bash update.sh --rollback'
    echo 'PANAINO_INSTALL_DIR overrides the directory containing this script.'
}
fail() { echo "Error: $*" >&2; exit 1; }
mode=update
tag=latest
for arg in "$@"; do
    case "$arg" in
        --help|-h) usage; exit 0 ;;
        --stage) [[ "$mode" == update ]] || fail 'Conflicting modes'; mode=stage ;;
        --rollback) [[ "$mode" == update ]] || fail 'Conflicting modes'; mode=rollback ;;
        latest|build-*) [[ "$tag" == latest ]] || fail 'Specify one release'; tag=$arg ;;
        *) fail 'Unknown argument' ;;
    esac
done

install_dir=${PANAINO_INSTALL_DIR:-$(cd "$(dirname "$0")" && pwd)}
[[ -d "$install_dir" ]] || fail 'Install directory does not exist'
install_dir=$(readlink -f "$install_dir")
config_file="$install_dir/config.json"
env_file="$install_dir/bot.env"
binary="$install_dir/panaino-bot"
previous="$install_dir/panaino-bot.previous"
service=panaino-bot.service
repo=aki-lua87/panaino_bot
for command in curl python3 sha256sum flock readlink mktemp; do
    command -v "$command" >/dev/null || fail "Missing command: $command"
done
[[ -f "$config_file" ]] || fail 'config.json is required'
exec 9>"$install_dir/.update.lock"
flock -n 9 || fail 'Another update is running'
mkdir -p "$install_dir/releases"
work=$(mktemp -d "$install_dir/.update.XXXXXX")
switching=false
old_target=''

service_command() {
    if (( EUID == 0 )); then systemctl "$@"; else sudo -n systemctl "$@"; fi
}

link_binary() {
    ln -sfn "$1" "$work/next-link"
    mv -Tf "$work/next-link" "$binary"
}

healthy() {
    # Check that systemd keeps the new process alive, and is running the chosen file.
    local count pid actual expected
    expected=$(readlink -f "$binary")
    for count in {1..10}; do
        sleep 1
        service_command is-active --quiet "$service" || return 1
        pid=$(service_command show -p MainPID "$service")
        pid=${pid#MainPID=}
        [[ "$pid" =~ ^[1-9][0-9]*$ ]] || return 1
        actual=$(readlink -f "/proc/$pid/exe") || return 1
        [[ "$actual" == "$expected" ]] || return 1
    done
}

cleanup() {
    local result=$?
    trap - EXIT INT TERM
    if [[ "$switching" == true ]]; then
        echo 'Update failed; restoring the previous binary.' >&2
        if link_binary "$old_target" && service_command restart "$service" && healthy; then
            echo 'Previous version restored.' >&2
        else
            echo 'Automatic recovery failed. Inspect panaino-bot.service and the previous binary.' >&2
        fi
    fi
    rm -rf -- "$work"
    exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

download() {
    curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
        --connect-timeout 10 --max-time 180 --retry 3 --output "$2" "$1"
}

check_candidate() {
    "$1" --config "$config_file" --check
    if [[ "$mode" == stage ]]; then return; fi
    [[ -f "$env_file" ]] || fail 'bot.env is required'
    # Parse the token without sourcing shell code, printing it, or putting it in argv.
    python3 - "$1" "$config_file" "$env_file" <<'PY'
import os, shlex, subprocess, sys
try:
    token = None
    with open(sys.argv[3], encoding='utf-8') as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith('#'):
                continue
            key, sep, value = line.partition('=')
            if not sep or key != 'DISCORD_TOKEN' or token is not None:
                raise ValueError()
            parts = shlex.split(value)
            if len(parts) != 1 or not parts[0]:
                raise ValueError()
            token = parts[0]
    if not token:
        raise ValueError()
    env = os.environ.copy()
    env['DISCORD_TOKEN'] = token
    sys.exit(subprocess.call([sys.argv[1], '--config', sys.argv[2], '--check-discord'], env=env))
except Exception:
    print('Could not validate bot.env/Discord settings (details suppressed).', file=sys.stderr)
    sys.exit(1)
PY
}

if [[ "$mode" != stage ]]; then
    service_command is-active --quiet "$service" || fail 'Service is not running. Use --stage for first-time setup.'
    [[ -x "$binary" ]] || fail 'Current binary is missing'
    old_target=$(readlink -f "$binary")
fi

if [[ "$mode" == rollback ]]; then
    [[ -x "$previous" ]] || fail 'No previous binary is saved'
    candidate=$(readlink -f "$previous")
    [[ "$candidate" == "$install_dir/releases/"* ]] || fail 'Previous binary is outside the release directory'
else
    if [[ "$tag" == latest ]]; then
        download "https://api.github.com/repos/$repo/releases/latest" "$work/release.json"
        tag=$(python3 - "$work/release.json" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as f:
    release = json.load(f)
if release.get('draft') or release.get('prerelease'):
    sys.exit('Release is not a stable published build.')
print(release['tag_name'])
PY
        )
    fi
    [[ "$tag" =~ ^build-[0-9a-f]{40}$ ]] || fail 'Release tag must be build- followed by a full commit SHA'
    base="https://github.com/$repo/releases/download/$tag"
    download "$base/panaino-bot" "$work/panaino-bot"
    download "$base/SHA256SUMS" "$work/SHA256SUMS"
    # Accept exactly our single binary entry, never arbitrary paths from a checksum file.
    checksum=$(python3 - "$work/SHA256SUMS" <<'PY'
import re, sys
with open(sys.argv[1], encoding='ascii') as f:
    content = f.read()
match = re.fullmatch(r'([0-9a-f]{64})  panaino-bot\n?', content)
if not match:
    sys.exit('Invalid checksum manifest.')
print(match.group(1))
PY
    )
    printf '%s  panaino-bot\n' "$checksum" > "$work/verified-checksum"
    (cd "$work" && sha256sum --check verified-checksum)
    chmod 700 "$work/panaino-bot"
    built_version=$("$work/panaino-bot" --version)
    [[ "$built_version" == "panaino-bot ${tag#build-} ("* ]] || fail 'Binary version does not match the release commit'
    check_candidate "$work/panaino-bot"
    release_dir="$install_dir/releases/$tag"
    if [[ -e "$release_dir" ]]; then
        [[ -f "$release_dir/panaino-bot" ]] || fail 'Existing release directory is incomplete'
        [[ "$(sha256sum "$release_dir/panaino-bot" | cut -d ' ' -f 1)" == "$checksum" ]] || fail 'Existing release binary differs from the published version'
    else
        mkdir "$release_dir"
        mv "$work/panaino-bot" "$release_dir/panaino-bot"
        cp "$work/SHA256SUMS" "$release_dir/SHA256SUMS"
    fi
    candidate="$release_dir/panaino-bot"
fi

if [[ "$mode" == stage ]]; then
    echo "Staged: $candidate"
    echo 'Current binary and running service have not been changed.'
    exit 0
fi
if [[ "$mode" == rollback ]]; then check_candidate "$candidate"; fi
if [[ "$candidate" == "$old_target" ]]; then echo 'Already running this version.'; exit 0; fi

# Keep a durable copy even when upgrading an earlier regular-file installation.
if [[ "$old_target" != "$install_dir/releases/"* ]]; then
    backup_dir="$install_dir/releases/local-$(date -u +%Y%m%dT%H%M%SZ)-$$"
    mkdir "$backup_dir"
    cp "$old_target" "$backup_dir/panaino-bot"
    chmod 700 "$backup_dir/panaino-bot"
    old_target="$backup_dir/panaino-bot"
fi
ln -s "$old_target" "$work/previous-link"
mv -Tf "$work/previous-link" "$previous"
switching=true
link_binary "$candidate"
service_command restart "$service"
healthy || fail 'The new process did not stay healthy'
switching=false
echo "Running: $candidate"
