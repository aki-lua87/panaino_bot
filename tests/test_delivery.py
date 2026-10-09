"""Exercise delivery failures with local fixtures; never contact GitHub or systemd."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
A = "a" * 40
B = "b" * 40


def executable(path, text):
    path.write_text(text, encoding="utf-8")
    path.chmod(0o700)


def binary(sha):
    return """#!/bin/bash
if [[ "$1" == --version ]]; then
  echo 'panaino-bot %s (mock linux/amd64)'
elif [[ "$3" == --check-discord ]]; then
  [[ "${FAIL_DISCORD:-0}" == 0 ]] || exit 1
elif [[ "$3" == --check ]]; then
  [[ "${FAIL_CONFIG:-0}" == 0 ]] || exit 1
else
  exit 1
fi
""" % sha


class UpdaterTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.install = self.root / "install"
        self.commands = self.root / "commands"
        self.assets = self.root / "assets"
        for path in (self.install, self.commands, self.assets):
            path.mkdir()
        shutil.copyfile(ROOT / "deploy/update.sh", self.install / "update.sh")
        (self.install / "config.json").write_text('{"guilds":{}}')
        (self.install / "bot.env").write_text("DISCORD_TOKEN=fake-private-token\n")
        self.log = self.root / "commands.log"
        self.env = os.environ.copy()
        self.env.update({
            "PATH": str(self.commands) + ":" + self.env["PATH"],
            "PANAINO_INSTALL_DIR": str(self.install),
            "FIXTURE_ROOT": str(self.root),
            "FIXTURE_INSTALL": str(self.install),
            "FIXTURE_LOG": str(self.log),
        })
        executable(self.commands / "curl", """#!/usr/bin/env python3
import json, os, pathlib, shutil, sys
args = sys.argv[1:]
url = args[-1]
out = pathlib.Path(args[args.index('--output') + 1])
with open(os.environ['FIXTURE_LOG'], 'a') as f: f.write(url + '\\n')
if os.environ.get('FAIL_DOWNLOAD') == '1': sys.exit(22)
if url.endswith('/releases/latest'):
    out.write_text(json.dumps({'tag_name': 'build-""" + B + """', 'draft':False,'prerelease':False}))
else:
    tag, asset = url.rsplit('/', 2)[-2:]
    shutil.copyfile(pathlib.Path(os.environ['FIXTURE_ROOT']) / 'assets' / tag / asset, out)
""")
        executable(self.commands / "systemctl", """#!/usr/bin/env python3
import os, pathlib, sys
args = sys.argv[1:]
target = pathlib.Path(os.environ['FIXTURE_INSTALL']) / 'panaino-bot'
with open(os.environ['FIXTURE_LOG'], 'a') as f: f.write('systemctl ' + ' '.join(args) + '\\n')
new = ('build-""" + B + """') in str(target.resolve())
if args[0] == 'restart' and new and os.environ.get('FAIL_RESTART') == '1': sys.exit(1)
if args[0] == 'is-active' and os.environ.get('INACTIVE') == '1': sys.exit(3)
if args[0] == 'is-active' and new and os.environ.get('FAIL_HEALTH') == '1': sys.exit(3)
if args[0] == 'show': print('MainPID=4242')
""")
        executable(self.commands / "sudo", "#!/bin/sh\n[ \"$1\" != -n ] || shift\nexec \"$@\"\n")
        executable(self.commands / "sleep", "#!/bin/sh\nexit 0\n")
        executable(self.commands / "readlink", """#!/bin/sh
if [ "$2" = /proc/4242/exe ]; then
  exec /usr/bin/readlink -f "$FIXTURE_INSTALL/panaino-bot"
fi
exec /usr/bin/readlink "$@"
""")
        for sha in (A, B):
            release = self.assets / ("build-" + sha)
            release.mkdir()
            executable(release / "panaino-bot", binary(sha))
            digest = hashlib.sha256((release / "panaino-bot").read_bytes()).hexdigest()
            (release / "SHA256SUMS").write_text(digest + "  panaino-bot\n")
        old = self.install / "releases" / ("build-" + A)
        old.mkdir(parents=True)
        shutil.copyfile(self.assets / ("build-" + A) / "panaino-bot", old / "panaino-bot")
        (old / "panaino-bot").chmod(0o700)
        (self.install / "panaino-bot").symlink_to(old / "panaino-bot")
        self.old = (old / "panaino-bot").resolve()

    def run_update(self, *args, **extra):
        env = dict(self.env, **extra)
        result = subprocess.run(["bash", str(self.install / "update.sh")] + list(args), env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        self.assertNotIn("fake-private-token", result.stdout)
        self.assertEqual((self.install / "config.json").read_text(), '{"guilds":{}}')
        self.assertEqual((self.install / "bot.env").read_text(), "DISCORD_TOKEN=fake-private-token\n")
        return result

    def test_stage_does_not_touch_current_or_service(self):
        result = self.run_update("--stage", INACTIVE="1")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual((self.install / "panaino-bot").resolve(), self.old)
        self.assertNotIn("systemctl", self.log.read_text())
        self.assertTrue((self.install / "releases" / ("build-" + B) / "panaino-bot").is_file())

    def test_update_and_manual_rollback(self):
        result = self.run_update()
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("build-" + B, str((self.install / "panaino-bot").resolve()))
        self.assertEqual((self.install / "panaino-bot.previous").resolve(), self.old)
        result = self.run_update("--rollback")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual((self.install / "panaino-bot").resolve(), self.old)

    def test_failure_modes_leave_or_restore_current(self):
        for flag in ("FAIL_DOWNLOAD", "FAIL_CONFIG", "FAIL_DISCORD", "FAIL_RESTART", "FAIL_HEALTH"):
            with self.subTest(flag=flag):
                result = self.run_update(**{flag:"1"})
                self.assertNotEqual(result.returncode, 0, result.stdout)
                self.assertEqual((self.install / "panaino-bot").resolve(), self.old)
        self.assertIn("Previous version restored", result.stdout)

    def test_bad_checksum_never_restarts(self):
        (self.assets / ("build-" + B) / "SHA256SUMS").write_text("0" * 64 + "  panaino-bot\n")
        result = self.run_update()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual((self.install / "panaino-bot").resolve(), self.old)
        self.assertNotIn("restart", self.log.read_text())

    def test_manifest_cannot_reference_other_files(self):
        (self.assets / ("build-" + B) / "SHA256SUMS").write_text("0" * 64 + "  ../bot.env\n")
        result = self.run_update()
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("restart", self.log.read_text())

    def test_release_and_binary_version_must_match(self):
        release = self.assets / ("build-" + B)
        executable(release / "panaino-bot", binary(A))
        digest = hashlib.sha256((release / "panaino-bot").read_bytes()).hexdigest()
        (release / "SHA256SUMS").write_text(digest + "  panaino-bot\n")
        result = self.run_update()
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("restart", self.log.read_text())

    def test_same_release_is_not_restarted(self):
        result = self.run_update("build-" + A)
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertNotIn("restart", self.log.read_text())

    def test_inactive_service_is_not_started(self):
        result = self.run_update(INACTIVE="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("restart", self.log.read_text())

    def test_downloads_are_pinned_to_one_tag(self):
        result = self.run_update("--stage")
        self.assertEqual(result.returncode, 0, result.stdout)
        urls = self.log.read_text().splitlines()
        self.assertEqual(len(urls), 3)
        self.assertTrue(all("/download/build-" + B + "/" in url for url in urls[1:]))

    def test_concurrent_update_is_rejected(self):
        import fcntl
        with open(self.install / ".update.lock", "w") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            result = self.run_update("--stage")
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(self.log.exists())


class PublishTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name in ("scripts", "dist", "commands"):
            (self.root / name).mkdir()
        shutil.copyfile(ROOT / "scripts/publish-release.sh", self.root / "scripts/publish-release.sh")
        executable(self.root / "dist/panaino-bot", binary(B))
        digest = hashlib.sha256((self.root / "dist/panaino-bot").read_bytes()).hexdigest()
        (self.root / "dist/SHA256SUMS").write_text(digest + "  panaino-bot\n")
        self.log = self.root / "gh.log"
        executable(self.root / "commands/gh", """#!/usr/bin/env python3
import os, sys
args = sys.argv[1:]
with open(os.environ['FIXTURE_LOG'], 'a') as f: f.write(' '.join(args) + '\\n')
if args[:2] == ['release', 'view']:
    status = os.environ.get('EXISTING', '')
    if not status: sys.exit(1)
    print(status)
elif args[0] == 'api' and '/git/ref/tags/' in args[1]: sys.exit(1)
elif args[0] == 'api' and '/git/ref/heads/master' in args[1]: print(os.environ['HEAD_SHA'])
elif args[:2] == ['release', 'upload'] and os.environ.get('FAIL_UPLOAD') == '1': sys.exit(1)
""")
        self.env = dict(os.environ, GITHUB_SHA=B, HEAD_SHA=B, GH_REPO="aki-lua87/panaino_bot", FIXTURE_LOG=str(self.log))
        self.env["PATH"] = str(self.root / "commands") + ":" + self.env["PATH"]

    def run_publish(self, **extra):
        return subprocess.run(["bash", str(self.root / "scripts/publish-release.sh")], env=dict(self.env, **extra), text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

    def test_assets_are_uploaded_before_publication(self):
        result = self.run_publish()
        self.assertEqual(result.returncode, 0, result.stdout)
        log = self.log.read_text()
        self.assertLess(log.index("release upload"), log.index("release edit"))
        self.assertIn("--draft", log)
        self.assertIn("--latest=true", log)

    def test_old_commit_cannot_replace_latest(self):
        result = self.run_publish(HEAD_SHA=A)
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("--latest=false", self.log.read_text())

    def test_incomplete_upload_stays_a_draft(self):
        result = self.run_publish(FAIL_UPLOAD="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("release edit", self.log.read_text())

    def test_successful_rerun_does_not_overwrite(self):
        result = self.run_publish(EXISTING="false")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertNotIn("release upload", self.log.read_text())

    def test_draft_can_be_resumed(self):
        result = self.run_publish(EXISTING="true")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertNotIn("release create", self.log.read_text())
        self.assertIn("release upload", self.log.read_text())


if __name__ == "__main__":
    unittest.main()
