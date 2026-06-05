#!/usr/bin/env bash
#
# End-to-end smoke test for the nikte CLI, run BEFORE cutting a Homebrew release.
# Drives the real binary against the live backend and asserts each flow, cleaning
# up everything it creates. Exits non-zero if any check fails.
#
# Usage:
#   scripts/smoke.sh [path-to-binary]     # default: ./nk-cli (build with `make build`)
#   NK=./nk-cli scripts/smoke.sh
#
# Requires: an authenticated session (`nk auth login`) and curl/python3.
set -u

NK="${1:-${NK:-./nk-cli}}"
API="https://auth.nikte.co"
PASS=0
FAIL=0
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

ok()   { echo "  ✓ $1"; PASS=$((PASS+1)); }
bad()  { echo "  ✗ $1"; FAIL=$((FAIL+1)); }
id_of(){ grep -oE 'ID: [a-zA-Z0-9]+' | tail -1 | awk '{print $2}'; }
url_of(){ grep -oE 'https://share\.nikte\.co/[a-zA-Z0-9]+' | head -1; }
http() { curl -s -o /dev/null -w '%{http_code}' -H 'Accept: application/json' -H 'User-Agent: nikte-smoke/1.0' "$1"; }

echo "nikte CLI smoke test — binary: $NK ($($NK --version 2>&1))"
echo

# 0) health + auth
[ "$($NK health 2>&1 | grep -c healthy)" = "1" ] && ok "health" || bad "health"

# 1) text round-trip
T="smoke text $(date +%s)"
ID=$($NK a "$T" 2>/dev/null | id_of)
GOT=$($NK g "$ID" 2>/dev/null | grep -F "$T")
[ -n "$GOT" ] && ok "text add/get ($ID)" || bad "text add/get"
$NK d "$ID" --force >/dev/null 2>&1

# 2) client-side encryption round-trip (text) — server must store ciphertext only
export NIKTE_PASSPHRASE='smoke-pass-123'
SECRET="encrypted secret $(date +%s)"
EID=$($NK a "$SECRET" --encrypt 2>/dev/null | id_of)
DEC=$($NK g "$EID" 2>/dev/null | grep -F "$SECRET")
WRONG=$(NIKTE_PASSPHRASE='nope' $NK g "$EID" 2>&1 | grep -c 'wrong passphrase')
[ -n "$DEC" ] && [ "$WRONG" = "1" ] && ok "encrypt text round-trip + wrong-pass fails ($EID)" || bad "encrypt text round-trip"
$NK d "$EID" --force >/dev/null 2>&1

# 3) encryption round-trip (file) — download to a separate dir and compare to original
echo "encrypted file payload $(date +%s)" > "$TMP/orig.txt"
mkdir -p "$TMP/dl"
FID=$($NK a "$TMP/orig.txt" --encrypt 2>/dev/null | id_of)
$NK g "$FID" -o "$TMP/dl" >/dev/null 2>&1
if diff -q "$TMP/orig.txt" "$TMP/dl/orig.txt" >/dev/null 2>&1; then ok "encrypt file round-trip ($FID)"; else bad "encrypt file round-trip"; fi
$NK d "$FID" --force >/dev/null 2>&1
unset NIKTE_PASSPHRASE

# 4) burn-after-read: 1st public view serves, 2nd is gone, underlying short deleted
BID=$($NK a "burn me $(date +%s)" 2>/dev/null | id_of)
BURL=$($NK p "$BID" --max-views 1 2>/dev/null | url_of)
C1=$(http "$BURL"); sleep 1; C2=$(http "$BURL")
GONE=$($NK g "$BID" 2>&1 | grep -c 'no item found')
[ "$C1" = "200" ] && [ "$C2" = "404" ] && [ "$GONE" = "1" ] && ok "burn-after-read (serve→burn→cascade) $BID" || bad "burn-after-read (got $C1 then $C2, gone=$GONE)"

# 5) view analytics: maxViews=5, two views → sh ls shows 2/5
AID=$($NK a "analytics $(date +%s)" 2>/dev/null | id_of)
AURL=$($NK p "$AID" --max-views 5 2>/dev/null | url_of)
http "$AURL" >/dev/null; http "$AURL" >/dev/null; sleep 1
[ "$($NK sh ls 2>/dev/null | grep "$AID" | grep -c '2/5')" = "1" ] && ok "view analytics (nk sh ls 2/5)" || bad "view analytics"
$NK d "$AID" --force >/dev/null 2>&1

# 6) URL shortener create → ls → delete
LURL=$($NK link "https://example.com/smoke-$(date +%s)" 2>/dev/null | grep -oE 'share\.nikte\.co/[a-zA-Z0-9]+')
CODE="${LURL##*/}"
[ -n "$CODE" ] && [ "$($NK link ls 2>/dev/null | grep -c "$CODE")" = "1" ] && ok "url shortener ($CODE)" || bad "url shortener"
$NK link d "$CODE" --force >/dev/null 2>&1

# 7) trustyou create → delete (web file request link)
RID=$($NK trustyou --title "smoke $(date +%s)" 2>/dev/null | grep -oE 'r/[a-zA-Z0-9]+' | head -1 | sed 's#r/##')
[ "$(http "$API/r/$RID")" = "200" ] && ok "trustyou link ($RID)" || bad "trustyou link"
TOKEN=$(python3 -c "import json,os;print(json.load(open(os.path.expanduser('~/Library/Application Support/nikte/config.json')))['id_token'])" 2>/dev/null)
[ -n "$TOKEN" ] && curl -s -o /dev/null -X DELETE "$API/request-links/$RID" -H "Authorization: Bearer $TOKEN"

echo
echo "──────────────────────────────────────"
echo "  PASS: $PASS   FAIL: $FAIL"
[ "$FAIL" -eq 0 ] && { echo "  ✅ smoke OK — safe to release"; exit 0; } || { echo "  ❌ smoke FAILED — do not release"; exit 1; }
