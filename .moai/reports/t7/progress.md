# t7 progress — C-g DWIM (Class B, lane-2)

## Cause-establishing evidence (reproduction)
- The card's literal ask already landed: `ee3f3fb` (2026-09-19) added `imoogi-keyboard-quit`,
  the global C-g binding and `tests/keyboard-quit-test.el`; it is on origin/main (7d98ec0).
- Full user config (tmux, `emacs -nw`, M-x -> C-x o -> C-g): minibuffer closed,
  `MB=nil C-g=imoogi-keyboard-quit`. The user-facing symptom does not reproduce on main.
  Gap: that run loaded `~/.emacs.d` (a real directory, not a symlink to this repo);
  `symbol-file` was not checked.
- Reproduced defect: `emacs -nw -Q` + the DWIM alone -> C-g printed
  `Symbol's function definition is void: minibuffer-keyboard-quit` and the minibuffer stayed open.
  `minibuffer-keyboard-quit` lives in `delsel`; it works on main only because
  `modules/general/11-editing.el:73` runs `(delete-selection-mode 1)`.
  The existing test mocks that function, so the suite could not see it.

## Decision record
- Scope narrowed from "implement C-g DWIM" (already on main) to "remove the hidden delsel dependency".
- Replacement: `abort-minibuffers`, the command Emacs 30.2 binds to C-g in
  `minibuffer-local-map` (measured: `map=abort-minibuffers`), so C-g from another window does
  what C-g inside the minibuffer does. Trade-off: it no longer deactivates an active region in
  the minibuffer before aborting (the delsel behaviour).

## Verification
- RED (before fix): new test `imoogi-quit-works-without-delete-selection-mode` failed with
  `"NO-DELSEL VOID:minibuffer-keyboard-quit"`.
- GREEN: `IMOOGI_ERT_SELECTOR=imoogi-quit ./tests/run.sh` -> exit 0 (check-parens + offline boot +
  ERT), `Ran 4 tests, 4 results as expected, 0 unexpected`.
- Interactive tmux, `-Q` + fixed 01-keys.el only: after C-g `"MB=nil delsel=nil"`.
- card-review: see card-review.md (codex, verdict pass, advisory).

## Residual risk / gaps
- GUI and separate-frame cases untested (tmux only). Nested minibuffers not exercised.
- Full ERT suite not run lane-locally (card-scoped per lane rules).
- `moai factory next --card t7` refused: `serial-slot: another serial card is in flight`.
  Work done under the leader's direct dispatch; no lease recorded.
- Commit trailer: session attribution policy changed mid-card to `Co-Authored-By`, so the
  dispatch's requested `🗿 MoAI` trailer was not used.
