from __future__ import annotations
from typing import Any

from mprutil.domain.unit import Unit, UnitId
from mprutil.domain.document import BsonDocument, BsonElement, BsonType
from mprutil.domain.reference import ByIdRef, ByNameRef
from mprutil.infrastructure.bson_parser import BsonParser
from mprutil.infrastructure.mpr_base import MprReader

_REF_KEYS = {
    "OriginPointer", "DestinationPointer",
    "TypePointer", "ValueTypeID", "PropertyTypeID",
    "ObjectTypeID",
}


class ReferenceAnalyzer:
    """引用追踪器"""

    def __init__(self, reader: MprReader) -> None:
        self._reader = reader

    def extract_element_types(self, data: bytes) -> list[tuple[str, str]]:
        """提取 (GUID-le-hex, $Type) 对"""
        results: list[tuple[str, str]] = []
        pos = 0
        while pos < len(data) - 20:
            if data[pos : pos + 5] == b"\x05$ID":
                blen = int.from_bytes(data[pos + 5 : pos + 9], "little", signed=True)
                if blen == 16:
                    guid = data[pos + 10 : pos + 26].hex()[:16]
                    # find $Type after this $ID
                    rest = data[pos + 26 : pos + 200]
                    tp_idx = rest.find(b"\x02$Type")
                    if tp_idx >= 0:
                        sl = int.from_bytes(
                            rest[tp_idx + 6 : tp_idx + 10], "little", signed=True
                        )
                        if 0 < sl < 200:
                            tp = rest[tp_idx + 10 : tp_idx + 10 + sl - 1].decode(
                                "ascii", errors="replace"
                            )
                            results.append((guid, tp))
                            pos += tp_idx + 10 + sl
                            continue
            pos += 1
        return results

    def find_dangling_refs(self, unit: Unit) -> list[ByIdRef]:
        """在 unit 中检测悬挂 Pointers"""
        defined = set()
        for el_id, _ in self.extract_element_types(unit.raw_bytes):
            defined.add(el_id)

        dangling: list[ByIdRef] = []
        for ref in self._extract_byid_refs(unit.raw_bytes):
            target_bytes = UnitId.uuid_to_guid_le(ref.target_id)
            target_hex = target_bytes.hex()[:16]
            if target_hex not in defined:
                dangling.append(ref)
        return dangling

    def _extract_byid_refs(self, data: bytes) -> list[ByIdRef]:
        """提取 ByIdRef 引用"""
        refs: list[ByIdRef] = []
        # Search for GUID patterns that might be ByIdRef values
        pos = 0
        while pos < len(data) - 30:
            if data[pos : pos + 5] == b"\x05$ID":
                blen = int.from_bytes(data[pos + 5 : pos + 9], "little", signed=True)
                if blen == 16:
                    # This is a $ID definition, skip
                    pos += 4 + blen + 1
                    continue
            # Look for binary (0x05) value that is 16 bytes (GUID)
            if data[pos] == 0x05:
                blen = int.from_bytes(data[pos + 1 : pos + 5], "little", signed=True)
                if blen == 16:
                    # Check if 5 bytes before is "$ID" — if so, it's a definition
                    prefix = data[max(0, pos - 6) : pos]
                    if prefix == b"\x05$ID\x00":
                        pos += 4 + blen + 1
                        continue
                    # It's a reference: find the key name
                    # Search backwards for key
                    key_start = pos - 1
                    while key_start > 0 and data[key_start] != 0x00 and data[key_start] != 0x03:
                        key_start -= 1
                    key = data[key_start + 1 : pos].decode(
                        "ascii", errors="replace"
                    ) if pos - key_start - 1 < 100 else "?"
                    guid_str = UnitId.guid_le_to_uuid(data[pos + 5 : pos + 21])
                    refs.append(ByIdRef(guid_str, key, pos))
                    pos += 4 + blen + 1
                    continue
            pos += 1
        return refs

    def build_ref_graph(self, unit: Unit) -> dict[str, set[str]]:
        graph: dict[str, set[str]] = {}
        for el_id, tp in self.extract_element_types(unit.raw_bytes):
            graph[el_id] = graph.get(el_id, set())
            graph[el_id].add(tp)
        for ref in self._extract_byid_refs(unit.raw_bytes):
            target_hex_raw = UnitId.uuid_to_guid_le(ref.target_id).hex()[:16]
            if target_hex_raw not in graph:
                graph[target_hex_raw] = set()
            graph[target_hex_raw].add(f"REF: {ref.source_key}")
        return graph

    def trace_refs(self, guid: str) -> tuple[Unit | None, list[Unit]]:
        """找出定义该 GUID 的 Unit 和引用它的 Units"""
        guid_bytes = UnitId.uuid_to_guid_le(guid)
        target: Unit | None = None
        referrers: list[Unit] = []
        for u in self._reader.iter_units():
            if guid_bytes in u.raw_bytes:
                doc = u.document
                id_elem = doc.find("$ID")
                if id_elem:
                    try:
                        if id_elem.as_guid() == guid:
                            target = u
                            continue
                    except ValueError:
                        pass
                referrers.append(u)
        return target, referrers

    def collect_refs_in_unit(
        self, data: bytes
    ) -> dict[str, list[dict[str, Any]]]:
        """收集 Unit 中所有 ByIdRef 引用，按引用类型分组"""
        result: dict[str, list[dict[str, Any]]] = {}
        pos = 0
        while pos < len(data) - 20:
            # Look for binary (0x05) followed by length 16
            if pos + 5 < len(data) and data[pos] == 0x05:
                blen = int.from_bytes(data[pos + 1 : pos + 5], "little", signed=True)
                if blen == 16:
                    prefix_check = max(0, pos - 6)
                    if data[prefix_check : prefix_check + 5] != b"\x05$ID":
                        key_start = data.rfind(b"\x00", max(0, pos - 50), pos)
                        key = data[key_start + 1 : pos].decode(
                            "ascii", errors="replace"
                        ) if key_start >= 0 else "?"
                        guid_str = UnitId.guid_le_to_uuid(data[pos + 5 : pos + 21])
                        info = {
                            "target": guid_str,
                            "offset": pos,
                            "key": key,
                        }
                        kind = key.split("_")[0] if "_" in key else key
                        result.setdefault(kind, []).append(info)
                    pos += 4 + 16 + 1
                    continue
            pos += 1
        return result
