#!/usr/bin/env python3
"""
create_demo_asset.py — Create a demo fieldset, model, and asset in Snipe-IT
for README screenshot purposes.

Usage:
    SNIPE_URL=https://your-instance.snipe-it.io \\
    SNIPE_KEY=your-api-key \\
    python3 .github/scripts/create_demo_asset.py

The script is idempotent: if a fieldset/model/asset with the same name already
exists it prints the existing ID and skips creation.

To clean up afterwards, run with --delete:
    python3 .github/scripts/create_demo_asset.py --delete
"""

import json
import os
import sys
import urllib.request
import urllib.error

SNIPE_URL = os.environ.get("SNIPE_URL", "").rstrip("/")
SNIPE_KEY = os.environ.get("SNIPE_KEY", "")

if not SNIPE_URL or not SNIPE_KEY:
    print("ERROR: set SNIPE_URL and SNIPE_KEY environment variables", file=sys.stderr)
    sys.exit(1)

HEADERS = {
    "Authorization": f"Bearer {SNIPE_KEY}",
    "Accept": "application/json",
    "Content-Type": "application/json",
}

# IDs that must already exist in the Snipe-IT instance
MANUFACTURER_ID = 1   # Apple
STATUS_ID       = 2   # Ready to Deploy
CATEGORY_ID     = 2   # Computers
SUPPLIER_ID     = 1   # CDW-G

# AXM custom field IDs created by `axm2snipe setup`
AXM_FIELD_IDS = [25, 26, 27, 28, 29, 30, 31, 32, 38, 45, 46, 47, 48]
# Built-in field IDs: MAC Address, RAM, Storage, Warranty End Date, Warranty ID, OS Version, Color
BUILTIN_FIELD_IDS = [1, 2, 3, 4, 5, 6, 7]

FIELDSET_NAME = "axm2snipe Demo"
MODEL_NAME    = "MacBook Pro M5 Pro"
MODEL_NUMBER  = "Mac16,8"
ASSET_TAG     = "DEMO-MBP-001"
SERIAL        = "C02ZR4XHMD6V"


def api(method, path, body=None):
    url = f"{SNIPE_URL}/api/v1{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, headers=HEADERS, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        print(f"HTTP {e.code} {method} {path}: {e.read().decode()}", file=sys.stderr)
        sys.exit(1)


def find_by_name(rows, name):
    for r in rows:
        if r.get("name") == name:
            return r
    return None


def create():
    # ── 1. Fieldset ───────────────────────────────────────────────────────────
    print(f"\n[1/3] Fieldset: {FIELDSET_NAME!r}")
    fs = find_by_name(api("GET", "/fieldsets")["rows"], FIELDSET_NAME)
    if fs:
        fieldset_id = fs["id"]
        print(f"  → already exists (id={fieldset_id}), skipping")
    else:
        result = api("POST", "/fieldsets", {"name": FIELDSET_NAME})
        if result.get("status") != "success":
            print(f"  ERROR: {result}", file=sys.stderr); sys.exit(1)
        fieldset_id = result["payload"]["id"]
        print(f"  → created (id={fieldset_id})")
        for fid in BUILTIN_FIELD_IDS + AXM_FIELD_IDS:
            r = api("POST", f"/fields/{fid}/associate", {"fieldset_id": fieldset_id})
            print(f"     associate field {fid}: {r.get('status', '?')}")

    # ── 2. Model ──────────────────────────────────────────────────────────────
    print(f"\n[2/3] Model: {MODEL_NAME!r}")
    mdl = find_by_name(api("GET", "/models?limit=500")["rows"], MODEL_NAME)
    if mdl:
        model_id = mdl["id"]
        print(f"  → already exists (id={model_id}), skipping")
    else:
        result = api("POST", "/models", {
            "name":            MODEL_NAME,
            "model_number":    MODEL_NUMBER,
            "manufacturer_id": MANUFACTURER_ID,
            "category_id":     CATEGORY_ID,
            "fieldset_id":     fieldset_id,
        })
        if result.get("status") != "success":
            print(f"  ERROR: {result}", file=sys.stderr); sys.exit(1)
        model_id = result["payload"]["id"]
        print(f"  → created (id={model_id})")

    # ── 3. Asset ──────────────────────────────────────────────────────────────
    print(f"\n[3/3] Asset: {ASSET_TAG!r} (serial {SERIAL})")
    existing = api("GET", f"/hardware/byserial/{SERIAL}")
    if existing.get("total", 0) > 0:
        asset_id = existing["rows"][0]["id"]
        print(f"  → already exists (id={asset_id}), skipping")
    else:
        notes = (
            "=== axm2snipe:warranty-start ===\n"
            "Status | Coverage | Start | End | Agreement | Payment\n"
            "Active | AppleCare+ for Mac | 2025-03-01 | 2028-03-01 | AC87654321 | Paid Up Front\n"
            "=== axm2snipe:warranty-end ==="
        )
        result = api("POST", "/hardware", {
            "asset_tag":      ASSET_TAG,
            "serial":         SERIAL,
            "name":           "MacBook Pro (Space Black)",
            "model_id":       model_id,
            "status_id":      STATUS_ID,
            "supplier_id":    SUPPLIER_ID,
            "purchase_date":  "2025-03-01",
            "purchase_cost":  "2499.00",
            "order_number":   "1CJ6QLW",
            "warranty_months": 36,
            "notes":          notes,
            "_snipeit_color_7":                       "Space Black",
            "_snipeit_storage_3":                     "512",
            "_snipeit_ram_2":                         "24",
            "_snipeit_mac_address_1":                 "2C:CA:16:4B:D2:9D",
            "_snipeit_axm_wi_fi_mac_address_48":      "2C:CA:16:4B:D2:9D",
            "_snipeit_axm_bluetooth_mac_address_45":  "2C:CA:16:4B:D2:9E",
            "_snipeit_axm_part_number_47":            "MX2Y3LL/A",
            "_snipeit_axm_applecare_status_30":       "Active",
            "_snipeit_axm_applecare_description_26":  "AppleCare+ for Mac",
            "_snipeit_axm_applecare_start_date_29":   "2025-03-01",
            "_snipeit_warranty_end_date_4":           "2028-03-01",
            "_snipeit_warranty_id_5":                 "AC87654321",
            "_snipeit_axm_applecare_renewable_28":    "true",
            "_snipeit_axm_applecare_payment_type_27": "Paid Up Front",
            "_snipeit_axm_assigned_mdm_server_31":    "CampusTech Jamf",
            "_snipeit_axm_mdm_assigned_38":           "1",
            "_snipeit_axm_added_to_org_25":           "2025-03-01",
        })
        if result.get("status") != "success":
            print(f"  ERROR: {result}", file=sys.stderr); sys.exit(1)
        asset_id = result["payload"]["id"]
        print(f"  → created (id={asset_id})")

    print(f"\nDone.")
    print(f"  Asset:    {SNIPE_URL}/hardware/{asset_id}")
    print(f"  Fieldset: {SNIPE_URL}/fields/fieldsets/{fieldset_id}/edit")


def delete():
    print("Deleting demo data...\n")

    # Delete asset by serial
    existing = api("GET", f"/hardware/byserial/{SERIAL}")
    if existing.get("total", 0) > 0:
        asset_id = existing["rows"][0]["id"]
        api("DELETE", f"/hardware/{asset_id}")
        print(f"  Deleted asset id={asset_id}")
    else:
        print("  Asset not found, skipping")

    # Delete model by name
    mdl = find_by_name(api("GET", "/models?limit=500")["rows"], MODEL_NAME)
    if mdl:
        api("DELETE", f"/models/{mdl['id']}")
        print(f"  Deleted model id={mdl['id']}")
    else:
        print("  Model not found, skipping")

    # Delete fieldset by name
    fs = find_by_name(api("GET", "/fieldsets")["rows"], FIELDSET_NAME)
    if fs:
        api("DELETE", f"/fieldsets/{fs['id']}")
        print(f"  Deleted fieldset id={fs['id']}")
    else:
        print("  Fieldset not found, skipping")

    print("\nDone.")


if "--delete" in sys.argv:
    delete()
else:
    create()
