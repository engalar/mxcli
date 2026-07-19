"""python -m mprutil <mpr_path> [command]"""
import sys
from mprutil import open_mpr, UnitExplorer, ReferenceAnalyzer


def main() -> None:
    args = sys.argv[1:]
    if not args:
        print("Usage: python -m mprutil <mpr|mprcontents> [command]")
        print("Commands: info, nanos, micros, types, dump <guid>")
        sys.exit(1)

    path = args[0]
    cmd = args[1] if len(args) > 1 else "info"

    with open_mpr(path) as reader:
        print(f"Project version: {reader.project_version}")
        explorer = UnitExplorer(reader)

        if cmd == "info":
            counts: dict[str, int] = {}
            for u in reader.iter_units():
                t = u.type_name or "?"
                counts[t] = counts.get(t, 0) + 1
            print(f"Total units: {sum(counts.values())}")
            for t, c in sorted(counts.items(), key=lambda x: -x[1])[:20]:
                print(f"  {t}: {c}")

        elif cmd == "nanos":
            for u, rv, rt in explorer.find_return_variables():
                if "Nanoflow" not in (u.type_name or ""):
                    continue
                info = explorer.describe(u)
                print(f"{u.id}: {info.get('name','?')} → ${rv}: {rt}")

        elif cmd == "micros":
            for u, rv, rt in explorer.find_return_variables():
                if "Microflow" not in (u.type_name or ""):
                    continue
                info = explorer.describe(u)
                print(f"{u.id}: {info.get('name','?')} → ${rv}: {rt}")

        elif cmd == "types":
            for u in reader.iter_units():
                t = u.type_name
                if t and t != "?":
                    print(f"{u.id}: {t}")

        elif cmd == "dump":
            if len(args) < 2:
                print("Need unit ID")
                return
            uid = args[2]
            u = reader.get_unit(uid)
            if u is None:
                print(f"Unit {uid} not found")
                return
            info = explorer.describe(u)
            for k, v in info.items():
                print(f"  {k}: {v}")
            strings = explorer.extract_strings(u)
            for off, s in strings[:30]:
                print(f"    @{off}: {s}")

        elif cmd == "refs":
            analyzer = ReferenceAnalyzer(reader)
            for u in reader.iter_units():
                dangling = analyzer.find_dangling_refs(u)
                if dangling:
                    print(f"{u.id} ({u.type_name}): {len(dangling)} dangling refs")
                    for r in dangling[:5]:
                        print(f"  → {r.target_id[:16]} ({r.source_key})")

        elif cmd == "elements":
            analyzer = ReferenceAnalyzer(reader)
            for u in reader.iter_units():
                if "Nanoflow" in (u.type_name or ""):
                    elements = analyzer.extract_element_types(u.raw_bytes)
                    if elements:
                        print(f"{u.id[:16]} ({len(elements)} elements):")
                        for el_id, tp in elements[:10]:
                            print(f"  {el_id}  {tp}")


if __name__ == "__main__":
    main()
