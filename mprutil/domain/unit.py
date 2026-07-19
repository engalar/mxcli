from __future__ import annotations
from dataclasses import dataclass, field
from typing import Iterator, TYPE_CHECKING

if TYPE_CHECKING:
    from mprutil.infrastructure.bson_parser import BsonDocument


@dataclass(frozen=True)
class UnitId:
    uuid: str

    def __str__(self) -> str:
        return self.uuid

    @classmethod
    def from_hex(cls, hex_str: str) -> UnitId:
        raw = hex_str.replace("-", "").lower()
        if len(raw) != 32:
            raise ValueError(f"invalid UUID hex: {hex_str!r}")
        return cls(f"{raw[:8]}-{raw[8:12]}-{raw[12:16]}-{raw[16:20]}-{raw[20:32]}")

    @staticmethod
    def guid_le_to_uuid(raw: bytes) -> str:
        if len(raw) != 16:
            raise ValueError(f"need 16 bytes, got {len(raw)}")
        a = int.from_bytes(raw[0:4], "little")
        b = int.from_bytes(raw[4:6], "little")
        c = int.from_bytes(raw[6:8], "little")
        d = raw[8:16]
        return (
            f"{a:08x}-{b:04x}-{c:04x}-{d[0]:02x}{d[1]:02x}-{d[2:].hex()}"
        )

    @staticmethod
    def uuid_to_guid_le(uuid_str: str) -> bytes:
        raw = uuid_str.replace("-", "")
        a = raw[0:8]; b = raw[8:12]; c = raw[12:16]
        rest = raw[16:32]
        return (
            int(a, 16).to_bytes(4, "little")
            + int(b, 16).to_bytes(2, "little")
            + int(c, 16).to_bytes(2, "little")
            + bytes.fromhex(rest)
        )


@dataclass
class Unit:
    id: UnitId
    container_id: bytes | None
    containment_name: str
    raw_bytes: bytes
    _document: object = field(repr=False, default=None)

    @property
    def document(self) -> BsonDocument:
        if self._document is None:
            from mprutil.infrastructure.bson_parser import BsonParser
            self._document = BsonParser.parse(self.raw_bytes)
        return self._document

    @property
    def type_name(self) -> str | None:
        d = self.document
        elem = d.find("$Type")
        return elem.as_string() if elem else None
