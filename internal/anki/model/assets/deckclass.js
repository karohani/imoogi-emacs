/* imoogi deck-class hook — REQ-C-006, design.md section 3.3.
 *
 * Runs inside the card each time it is shown. Anki expands {{Deck}} as the
 * full deck path — `::` separators, spaces, parentheses and all — and a CSS
 * class cannot carry that verbatim. So the template emits its wrapper with
 * the stable class `imoogi-deck` and the deck path as the text of a hidden
 * carrier element, and this script ADDS `deck-<token>` to the wrapper, where
 * <token> is the path normalized by exactly the seven steps
 * internal/anki/model/deckclass.go implements. The Go function is the
 * reference; the mirror test in deckclass_script_test.go runs this file
 * under node over the acceptance rows and fails when the two disagree.
 * A user styling the deck (PROGRAMMER)::(GO) writes a rule on the class
 * deck-programmer-go, as design.md section 3.4 promises.
 *
 * Runtime facts this file is written against:
 *  - Anki injects the card HTML into an existing page, so there is no load
 *    event to wait for; the script runs when the card is shown.
 *  - The Basic back side embeds {{FrontSide}}, so there this script runs
 *    twice and sees two carriers (the back's own and the front's, nested).
 *    Everything below is idempotent: a second run adds nothing new.
 *  - document.currentScript is not relied on; carriers are found by class.
 *  - The deck name travels as text read through textContent, so a double
 *    quote in a deck name cannot break out of an attribute.
 *
 * ES5 only, on purpose: it runs in Qt WebEngine, Android WebView and WKWebView.
 */
(function () {
  "use strict";

  function normalizeDeckClass(deck) {
    var i, c, s, out, last, token;

    /* Step 1 — ASCII case fold. Only A-Z are folded, never toLowerCase(),
     * which would fold non-ASCII letters that step 3 must turn into hyphens.
     */
    s = "";
    for (i = 0; i < deck.length; i++) {
      c = deck.charCodeAt(i);
      s += (c >= 65 && c <= 90) ? String.fromCharCode(c + 32) : deck.charAt(i);
    }

    /* Step 2 — each `::` separator becomes one hyphen, before step 3 so a
     * separator yields one hyphen rather than two.
     */
    s = s.split("::").join("-");

    /* Steps 3 and 4 — every code unit outside [a-z0-9_] becomes a hyphen,
     * runs collapsing as they are produced. Go maps runes where this maps
     * UTF-16 code units; a non-BMP rune is two units here and so two hyphens,
     * which step 4 collapses to the one hyphen Go emits — the results agree.
     */
    out = "";
    last = false;
    for (i = 0; i < s.length; i++) {
      c = s.charCodeAt(i);
      if ((c >= 97 && c <= 122) || (c >= 48 && c <= 57) || c === 95) {
        out += s.charAt(i);
        last = false;
      } else if (!last) {
        out += "-";
        last = true;
      }
    }

    /* Step 5 — strip the edges. */
    token = out.replace(/^-+/, "").replace(/-+$/, "");

    /* Step 7 before step 6, as in Go: an empty token has no first character
     * to test, and the two are disjoint.
     */
    if (token === "") {
      return "unnamed";
    }

    /* Step 6 — the digit guard. */
    c = token.charCodeAt(0);
    if (c >= 48 && c <= 57) {
      return "_" + token;
    }
    return token;
  }

  var carriers = document.querySelectorAll(".imoogi-deck-name");
  for (var i = 0; i < carriers.length; i++) {
    var wrapper = carriers[i].parentNode;
    if (!wrapper || !wrapper.classList) {
      continue;
    }
    wrapper.classList.add("deck-" + normalizeDeckClass(carriers[i].textContent || ""));
  }
})();
