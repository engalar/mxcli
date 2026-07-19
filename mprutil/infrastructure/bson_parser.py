from __future__ import annotations
import struct
from typing import Any

from mprutil.domain.document import BsonDocument, BsonElement

_BSON_TYPES = {0x01, 0x02, 0x03, 0x04, 0x05, 0x08, 0x10, 0x12}


class BsonParser:

    @staticmethod
    def parse(buffer: bytes, offset: int = 0) -> BsonDocument:
        """零拷贝解析顶层 BSON 文档"""
        if len(buffer) - offset < 5:
            raise ValueError(f"buffer too short at offset {offset}")
        length = struct.unpack_from("<i", buffer, offset)[0]
        if offset + length > len(buffer):
            raise ValueError(
                f"document length {length} exceeds buffer ({len(buffer) - offset} available) "
                f"at offset {offset}"
            )
        view = memoryview(buffer)
        elements, _ = BsonParser._parse_elements(view, offset + 4, offset + length)
        return BsonDocument(buffer=view, offset=offset, length=length, _elements=elements)

    @staticmethod
    def _parse_elements(
        view: memoryview, start: int, end: int
    ) -> tuple[list[BsonElement], int]:
        elements: list[BsonElement] = []
        pos = start
        while pos < end - 1:
            typ = view[pos]
            if typ == 0x00:
                pos += 1
                break
            if typ not in _BSON_TYPES:
                break
            pos += 1
            buf_bytes = bytes(view[pos:pos+200])
            null_pos = buf_bytes.find(b"\x00")
            if null_pos < 0:
                break
            key = buf_bytes[:null_pos].decode("ascii", errors="replace")
            pos += null_pos + 1
            val_offset = pos
            val_size = BsonParser._value_size(view, typ, pos)
            if val_size < 0:
                break
            pos += val_size
            elements.append(
                BsonElement(
                    typ=typ,
                    key=key,
                    offset=val_offset,
                    buffer=view,
                )
            )
        return elements, pos

    @staticmethod
    def _value_size(view: memoryview, typ: int, pos: int) -> int:
        if typ == 0x02:
            if pos + 4 > len(view):
                return -1
            slen = struct.unpack_from("<i", view, pos)[0]
            if slen <= 0:
                return -1
            return 4 + slen
        elif typ == 0x03 or typ == 0x04:
            if pos + 4 > len(view):
                return -1
            doc_len = struct.unpack_from("<i", view, pos)[0]
            if doc_len <= 0 or pos + doc_len > len(view):
                return -1
            return doc_len
        elif typ == 0x05:
            if pos + 4 > len(view):
                return -1
            blen = struct.unpack_from("<i", view, pos)[0]
            return 4 + blen + 1
        elif typ == 0x08:
            return 1
        elif typ == 0x10:
            return 4
        elif typ == 0x12:
            return 8
        elif typ == 0x01:
            return 8
        return -1

    @staticmethod
    def _parse_array(buffer: bytes) -> list[Any]:
        """解析 BSON 数组 → list of BsonDocument / primitive values"""
        arr_len = struct.unpack_from("<i", buffer, 0)[0]
        view = memoryview(buffer[:arr_len])
        items: list[Any] = []
        pos = 4
        while pos < arr_len - 1:
            typ = view[pos]
            if typ == 0x00:
                break
            if typ not in _BSON_TYPES:
                break
            pos += 1
            b2 = bytes(view[pos:pos+200])
            null_pos = b2.find(b"\x00")
            if null_pos < 0:
                break
            pos += null_pos + 1
            val_start = pos
            vsize = BsonParser._value_size(view, typ, pos)
            if vsize < 0:
                break
            if typ == 0x03:
                raw = bytes(view[pos : pos + vsize])
                items.append(BsonParser.parse(raw, 0))
            elif typ == 0x02:
                slen = struct.unpack_from("<i", view, pos)[0]
                items.append(bytes(view[pos + 4 : pos + 4 + slen - 1]).decode("utf-8", errors="replace"))
            elif typ == 0x10:
                items.append(struct.unpack_from("<i", view, pos)[0])
            elif typ == 0x12:
                items.append(struct.unpack_from("<q", view, pos)[0])
            elif typ == 0x08:
                items.append(view[pos] != 0)
            else:
                items.append(None)
            pos += vsize
        return items

    @staticmethod
    def extract_strings(
        buffer: bytes, keywords: set[str] | None = None
    ) -> list[tuple[int, str]]:
        """提取 BSON 中所有可读字符串"""
        results: list[tuple[int, str]] = []
        i = 0
        while i < len(buffer) - 2:
            if 32 <= buffer[i] < 127:
                end = buffer.find(b"\x00", i)
                if end > i and 2 <= (end - i) <= 200:
                    s = buffer[i:end].decode("ascii", errors="replace")
                    if s.isprintable():
                        if keywords is None or any(k in s for k in keywords):
                            results.append((i, s))
                i = end + 1
            else:
                i += 1
        return results

    @staticmethod
    def read_string(buffer: bytes, pos: int) -> tuple[str, int]:
        """读取 BSON string 值，返回 (value, next_offset)"""
        slen = struct.unpack_from("<i", buffer, pos)[0]
        val = buffer[pos + 4 : pos + 4 + slen - 1].decode("utf-8", errors="replace")
        return val, pos + 4 + slen

    @staticmethod
    def read_guid(buffer: bytes, pos: int) -> tuple[str, int]:
        """读取 BSON binary UUID，返回 (hex, next_offset)"""
        blen = struct.unpack_from("<i", buffer, pos)[0]
        if blen == 16:
            from mprutil.domain.unit import UnitId
            return UnitId.guid_le_to_uuid(buffer[pos + 5 : pos + 21]), pos + 4 + blen + 1
        return f"bin({blen})", pos + 4 + blen + 1
