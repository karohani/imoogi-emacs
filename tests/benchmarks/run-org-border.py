"""Measure the Org border prototype in disposable GUI Emacs processes."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--emacs", required=True, help="GUI Emacs executable")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--timeout", type=float, default=60)
    parser.add_argument("--variants", nargs="+", default=["baseline", "cached", "parser-only", "render-only", "bounded"],
                        choices=["baseline", "cached", "parser-only", "render-only", "bounded"])
    parser.add_argument("--sizes", nargs="+", type=int, default=[10000, 100000, 1000000])
    parser.add_argument("--without-org-menu", action="store_true")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    if any(args.output.glob("*.json")):
        parser.error("Use a new output directory to avoid stale partial results")
    args.output.mkdir(parents=True, exist_ok=True)
    output = args.output.resolve()
    statuses = []
    for shape in ("many-headings", "long-subtree"):
        for lines in args.sizes:
            for order, names in (("forward", args.variants),
                                 ("reverse", list(reversed(args.variants)))):
                variants = "(" + " ".join(names) + ")"
                name = f"{shape}-{lines}-{order}"
                with tempfile.TemporaryDirectory(prefix="imoogi-border-") as home:
                    env = dict(os.environ, BENCH_ROOT=str(root), BENCH_HOME=home,
                               BENCH_OUTPUT=str(output / f"{name}.json"),
                               BENCH_SHAPE=shape, BENCH_LINES=str(lines),
                               BENCH_VARIANTS=variants,
                               BENCH_NO_ORG_MENU="1" if args.without_org_menu else "0")
                    start = time.monotonic()
                    with (output / f"{name}.log").open("w") as log:
                        proc = subprocess.Popen(
                            [args.emacs, "-Q", "-l", str(root / "tests/benchmarks/org-border-isolated.el")],
                            env=env, stdout=log, stderr=log, start_new_session=True)
                        try:
                            status = proc.wait(timeout=args.timeout)
                        except subprocess.TimeoutExpired:
                            # Only this newly created benchmark process is killed.
                            proc.kill()
                            proc.wait()
                            status = "timeout"
                    row = dict(case=name, status=status,
                               elapsed_seconds=round(time.monotonic() - start, 3))
                    statuses.append(row)
                    (output / "status.json").write_text(json.dumps(statuses, indent=2))
                    print(json.dumps(row), flush=True)
    if any(row["status"] != 0 for row in statuses):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
