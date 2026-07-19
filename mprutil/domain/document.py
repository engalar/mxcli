from __future__ import annotations
from dataclasses import dataclass, field
from enum import IntEnum
from typing import Iterator


class BsonType(IntEnum):
    DOUBLE = 0x01
    STRING = 0x02
    DOCUMENT = 0x03
    ARRAY = 0x04
    BINARY = 0x05
    BOOL = 0x08
    INT32 = 0x10
    INT64 = 0x12


@dataclass
class BsonElement:
    typ: int
    key: str
    offset: int
    buffer: memoryview = field(repr=False)
    _val_cache: object = field(default=None, repr=False)

    def _value(self) -> bytes:
        return bytes(self.buffer[self.offset:])

    def as_string(self) -> str:
        from mprutil.infrastructure.bson_parser import BsonParser
        raw = self._value()
        slen = int.from_bytes(raw[:4], "little", signed=True)
        return raw[4 : 4 + slen - 1].decode("utf-8", errors="replace")

    def as_int32(self) -> int:
        return int.from_bytes(self._value()[:4], "little", signed=True)

    def as_int64(self) -> int:
        return int.from_bytes(self._value()[:8], "little", signed=True)

    def as_bool(self) -> bool:
        return self._value()[0] != 0

    def as_guid(self) -> str:
        raw = self._value()
        bin_len = int.from_bytes(raw[:4], "little", signed=True)
        if bin_len != 16:
            return f"bin({bin_len})"
        from mprutil.domain.unit import UnitId
        return UnitId.guid_le_to_uuid(raw[5:21])

    def as_document(self) -> BsonDocument:
        from mprutil.infrastructure.bson_parser import BsonParser
        raw = self._value()
        doc_len = int.from_bytes(raw[:4], "little", signed=True)
        return BsonParser.parse(bytes(self.buffer), self.offset)

    def as_array(self) -> list[object]:
        from mprutil.infrastructure.bson_parser import BsonParser
        return BsonParser._parse_array(self._value())

    def type_name(self) -> str | None:
        if self.typ != BsonType.DOCUMENT:
            return None
        doc = self.as_document()
        elem = doc.find("$Type")
        return elem.as_string() if elem else None


@dataclass
class BsonDocument:
    buffer: memoryview = field(repr=False)
    offset: int
    length: int
    _elements: list[BsonElement] = field(default_factory=list, repr=False)

    def __getitem__(self, key: str) -> BsonElement | None:
        for e in self._elements:
            if e.key == key:
                return e
        return None

    def find(self, key: str) -> BsonElement | None:
        return self[key]

    def __iter__(self) -> Iterator[BsonElement]:
        return iter(self._elements)

    def __len__(self) -> int:
        return len(self._elements)

    def iter_docs(self, key: str) -> Iterator[BsonDocument]:
        elem = self.find(key)
        if elem is None:
            return
        if elem.typ == BsonType.DOCUMENT:
            yield elem.as_document()
        elif elem.typ == BsonType.ARRAY:
            for item in elem.as_array():
                if isinstance(item, BsonDocument):
                    yield item
