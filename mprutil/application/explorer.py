from __future__ import annotations
from typing import Iterator

from mprutil.domain.unit import Unit, UnitId
from mprutil.infrastructure.bson_parser import BsonParser
from mprutil.infrastructure.mpr_base import MprReader


class UnitExplorer:
    """Unit 导航和搜索"""

    def __init__(self, reader: MprReader) -> None:
        self._reader = reader

    def find_by_type(self, type_name: str) -> list[Unit]:
        return list(self._reader.iter_units(type_filter=type_name))

    def find_by_content(self, pattern: bytes) -> list[Unit]:
        results: list[Unit] = []
        for u in self._reader.iter_units():
            if pattern in u.raw_bytes:
                results.append(u)
        return results

    def find_by_id(self, guid: str | bytes) -> list[tuple[Unit, int]]:
        """搜索引用特定 GUID 的 Unit"""
        if isinstance(guid, str):
            guid_bytes = UnitId.uuid_to_guid_le(guid)
        else:
            guid_bytes = guid
        results: list[tuple[Unit, int]] = []
        for u in self._reader.iter_units():
            pos = u.raw_bytes.find(guid_bytes)
            if pos >= 0:
                results.append((u, pos))
        return results

    def find_return_variables(self) -> Iterator[tuple[Unit, str, str]]:
        """yield (unit, return_var_name, type_name)"""
        types = {"Microflows$Microflow", "Microflows$Nanoflow", "Microflows$Rule"}
        for u in self._reader.iter_units():
            t = u.type_name
            if t not in types:
                continue
            doc = u.document
            rv = doc.find("ReturnVariableName")
            if rv is None:
                continue
            rv_val = rv.as_string()
            if not rv_val:
                continue
            rt = doc.find("MicroflowReturnType")
            rt_type = ""
            if rt:
                sub = rt.as_document()
                st = sub.find("$Type")
                if st:
                    rt_type = st.as_string()
            yield u, rv_val, rt_type

    def extract_strings(
        self, unit: Unit, keywords: set[str] | None = None
    ) -> list[tuple[int, str]]:
        return BsonParser.extract_strings(unit.raw_bytes, keywords)

    def count_activities(self, unit: Unit) -> dict[str, int]:
        counts: dict[str, int] = {}
        for offset, s in self.extract_strings(
            unit, keywords={"Microflows$", "Pages$", "Forms$", "CustomWidgets$"}
        ):
            if not s.startswith("Microflows$") and not s.startswith("Pages$"):
                continue
            if "$Type" not in s and s.count("$") >= 1:
                counts[s] = counts.get(s, 0) + 1
        return counts

    def describe(self, unit: Unit) -> dict[str, object]:
        d = unit.document
        info: dict[str, object] = {
            "id": str(unit.id),
            "type": unit.type_name or "?",
            "size": len(unit.raw_bytes),
        }
        name_el = d.find("Name")
        if name_el:
            info["name"] = name_el.as_string()
        rv = d.find("ReturnVariableName")
        if rv and rv.as_string():
            info["return_variable"] = rv.as_string()
        rt = d.find("ReturnType")
        if rt and rt.as_string():
            info["return_type"] = rt.as_string()
        info["activities"] = self.count_activities(unit)
        return info
