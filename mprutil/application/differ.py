from __future__ import annotations
from dataclasses import dataclass, field
from typing import Any, Iterator

from mprutil.domain.unit import Unit, UnitId
from mprutil.domain.document import BsonDocument, BsonElement, BsonType
from mprutil.infrastructure.bson_parser import BsonParser


@dataclass
class DiffEntry:
    kind: str  # "added" | "removed" | "changed" | "id-only"
    path: str
    expected: str | None = None
    actual: str | None = None


@dataclass
class UnitDiff:
    unit_id_a: str
    unit_id_b: str
    entries: list[DiffEntry] = field(default_factory=list)

    @property
    def structural_diffs(self) -> list[DiffEntry]:
        """排除 UUID 差异后的结构差异"""
        return [e for e in self.entries if e.kind != "id-only"]


class BsonDiffer:

    @staticmethod
    def diff_documents(a: BsonDocument, b: BsonDocument) -> list[DiffEntry]:
        """逐字段对比两个 BSON 文档（忽略 UUID）"""
        entries: list[DiffEntry] = []

        a_map = {e.key: e for e in a}
        b_map = {e.key: e for e in b}

        a_keys = set(a_map)
        b_keys = set(b_map)

        for key in sorted(a_keys - b_keys):
            entries.append(DiffEntry("removed", key, _short(a_map[key]), None))

        for key in sorted(b_keys - a_keys):
            entries.append(DiffEntry("added", key, None, _short(b_map[key])))

        for key in sorted(a_keys & b_keys):
            ae = a_map[key]
            be = b_map[key]
            if ae.typ != be.typ:
                entries.append(
                    DiffEntry(
                        "changed",
                        key,
                        f"type={ae.typ:02x}",
                        f"type={be.typ:02x}",
                    )
                )
                continue
            sa = _short(ae)
            sb = _short(be)
            if sa != sb:
                kind = "id-only" if _diff_only_ids(sa, sb) else "changed"
                if kind != "id-only":
                    entries.append(DiffEntry(kind, key, sa, sb))

        return entries

    @staticmethod
    def diff_units(u1: Unit, u2: Unit) -> UnitDiff:
        """对比两个 Unit，忽略 UUID 差异"""
        diff = UnitDiff(str(u1.id), str(u2.id))

        if u1.type_name != u2.type_name:
            diff.entries.append(
                DiffEntry("changed", "$Type", u1.type_name, u2.type_name)
            )
            return diff

        for entry in BsonDiffer.diff_documents(u1.document, u2.document):
            diff.entries.append(entry)

        return diff


def _short(elem: BsonElement) -> str:
    try:
        if elem.typ == BsonType.STRING:
            return elem.as_string()
        elif elem.typ in (BsonType.INT32,):
            return str(elem.as_int32())
        elif elem.typ == BsonType.BOOL:
            return str(elem.as_bool())
        elif elem.typ == BsonType.BINARY:
            raw = elem._value()
            blen = int.from_bytes(raw[:4], "little", signed=True)
            if blen == 16:
                return f"GUID({elem.as_guid()[:8]}...)"
            return f"bin({blen})"
        elif elem.typ == BsonType.DOCUMENT:
            tn = elem.type_name()
            if tn:
                return f"{{{tn}}}"
            return "{...}"
        elif elem.typ == BsonType.ARRAY:
            return "[...]"
        return f"<{elem.typ:02x}>"
    except Exception:
        return "<err>"


def _diff_only_ids(a: str, b: str) -> bool:
    """检查两个字符串的差异是否仅仅是 GUID 值不同"""
    import re
    def mask_guids(s: str) -> str:
        return re.sub(
            r"[0-9a-f]{8}\.\.\.",
            "<GUID>",
            s
        )
    return mask_guids(a) == mask_guids(b)
