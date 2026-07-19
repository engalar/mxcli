from __future__ import annotations
from dataclasses import dataclass


@dataclass
class ByIdRef:
    target_id: str
    source_key: str
    source_offset: int


@dataclass
class ByNameRef:
    qualified_name: str
    source_key: str
    source_offset: int
