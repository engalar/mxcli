#!/usr/bin/env python3
"""Key-based TextTemplate oracle for CE0463 debugging.

For a v1 .mpr, locate every CustomWidgets$CustomWidget instance, join each
Object property's TypePointer -> PropertyType.$ID -> PropertyKey, and record the
TextTemplate state (null / document / absent) of each property's Value, keyed by
PropertyKey (NOT fragile numeric index). Handles nested ObjectType (columns).

Usage: tt_oracle.py <built.mpr> <golden.mpr>
"""
import sqlite3, bson, binascii, sys

def hx(b):
    return binascii.hexlify(b).decode() if isinstance(b, (bytes, bytearray)) else str(b)

def load_page_docs(path):
    conn = sqlite3.connect(path)
    cur = conn.cursor()
    cur.execute("select Contents from Unit")
    for (blob,) in cur.fetchall():
        if not blob:
            continue
        try:
            yield bson.loads(blob)
        except Exception:
            continue

def find_widgets(o, out):
    if isinstance(o, dict):
        if o.get("$Type") == "CustomWidgets$CustomWidget" and "Type" in o and "Object" in o:
            out.append(o)
        for v in o.values():
            find_widgets(v, out)
    elif isinstance(o, list):
        for v in o:
            find_widgets(v, out)

def tt_state(val):
    if not isinstance(val, dict):
        return "n/a"
    if "TextTemplate" not in val:
        return "absent"
    tt = val["TextTemplate"]
    if tt is None:
        return "null"
    if isinstance(tt, dict):
        return "document"
    return type(tt).__name__

def build_id_key_map(property_types):
    """id(hex) -> PropertyKey, plus id(hex) -> nested PropertyTypes list (for Object props)."""
    id2key, id2nested = {}, {}
    for e in property_types:
        if not isinstance(e, dict):
            continue
        pid = hx(e.get("$ID"))
        key = e.get("PropertyKey") or e.get("Key")
        id2key[pid] = key
        vt = e.get("ValueType") if isinstance(e.get("ValueType"), dict) else {}
        nested_ot = vt.get("ObjectType") if isinstance(vt.get("ObjectType"), dict) else None
        if nested_ot and isinstance(nested_ot.get("PropertyTypes"), list):
            id2nested[pid] = nested_ot["PropertyTypes"]
    return id2key, id2nested

def build_id_ttype_map(property_types):
    """id(hex) -> True if ValueType.Type == 'TextTemplate' (top-level + nested)."""
    id2tt = {}
    for e in property_types:
        if not isinstance(e, dict):
            continue
        pid = hx(e.get("$ID"))
        vt = e.get("ValueType") if isinstance(e.get("ValueType"), dict) else {}
        id2tt[pid] = (vt.get("Type") == "TextTemplate")
    return id2tt

def collect(widget, tt_only=False):
    """Return {qualified_key: tt_state} for a widget.
    If tt_only, restrict to properties whose PropertyType.ValueType.Type == TextTemplate."""
    ot = widget["Type"]["ObjectType"]
    pts = ot.get("PropertyTypes") or []
    id2key, id2nested = build_id_key_map(pts)
    id2tt = build_id_ttype_map(pts)

    result = {}
    obj = widget["Object"]
    for prop in obj.get("Properties") or []:
        if not isinstance(prop, dict):
            continue
        tp = hx(prop.get("TypePointer"))
        key = id2key.get(tp, f"?{tp[:8]}")
        val = prop.get("Value")
        # nested ObjectType (e.g. columns): Value.Objects[] each has Properties[]
        if tp in id2nested and isinstance(val, dict) and isinstance(val.get("Objects"), list):
            nid2key, _ = build_id_key_map(id2nested[tp])
            nid2tt = build_id_ttype_map(id2nested[tp])
            for oi, colobj in enumerate(val["Objects"]):
                if not isinstance(colobj, dict):
                    continue
                for cprop in colobj.get("Properties") or []:
                    if not isinstance(cprop, dict):
                        continue
                    ctp = hx(cprop.get("TypePointer"))
                    if tt_only and not nid2tt.get(ctp):
                        continue
                    ckey = nid2key.get(ctp, f"?{ctp[:8]}")
                    result[f"{key}[{oi}].{ckey}"] = tt_state(cprop.get("Value"))
        else:
            if tt_only and not id2tt.get(tp):
                continue
            result[key] = tt_state(val)
    return result

def oracle(path, tt_only=False):
    widgets = []
    for doc in load_page_docs(path):
        find_widgets(doc, widgets)
    merged = {}
    for w in widgets:
        wid = w.get("Type", {}).get("WidgetId") or w.get("Type", {}).get("WidgetName") or "?"
        for k, v in collect(w, tt_only=tt_only).items():
            merged[f"{wid}::{k}"] = v
    return merged

def hidden_set(path):
    """Per widget-type, the COMPLETE golden state of every textTemplate-TYPE property.
    Collapses instance/column indices to reveal the stable hidden-set."""
    import re
    from collections import defaultdict
    g = oracle(path, tt_only=True)
    per = defaultdict(lambda: defaultdict(set))
    for k, state in g.items():
        wid, _, key = k.partition("::")
        key = re.sub(r"\[\d+\]", "[]", key)  # collapse indices
        per[wid][key].add(state)
    for wid in sorted(per):
        print(f"\n### {wid}")
        for key in sorted(per[wid]):
            states = per[wid][key]
            tag = "HIDDEN(null)" if states == {"null"} else ("ABSENT" if states == {"absent"} else ("VISIBLE(document)" if states == {"document"} else f"MIXED{sorted(states)}"))
            print(f"    {key:40} {tag}")

def main():
    if len(sys.argv) >= 3 and sys.argv[1] == "--hidden-set":
        hidden_set(sys.argv[2])
        return
    built, golden = sys.argv[1], sys.argv[2]
    b = oracle(built)
    g = oracle(golden)
    allkeys = sorted(set(b) | set(g))
    print(f"{'KEY':70} {'BUILT':10} {'GOLDEN':10}  DIFF")
    diffs = 0
    for k in allkeys:
        bv = b.get(k, "MISSING")
        gv = g.get(k, "MISSING")
        mark = "" if bv == gv else "  <<< DIFF"
        if mark:
            diffs += 1
        # only print TextTemplate-relevant rows (state in the tt vocabulary) OR diffs
        if bv in ("null", "document", "absent", "MISSING") or gv in ("null", "document", "absent", "MISSING"):
            if mark or bv in ("null", "document") or gv in ("null", "document"):
                print(f"{k:70} {bv:10} {gv:10}{mark}")
    print(f"\nTOTAL DIFFS: {diffs}")

if __name__ == "__main__":
    main()
