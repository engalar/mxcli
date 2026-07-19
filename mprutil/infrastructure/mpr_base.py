from __future__ import annotations
from typing import Iterator, Protocol, runtime_checkable

from mprutil.domain.unit import Unit


@runtime_checkable
class MprReader(Protocol):
    """统一 MPR v1/v2 读取接口"""

    def iter_units(self, type_filter: str | None = None) -> Iterator[Unit]:
        ...

    def get_unit(self, uid: str) -> Unit | None:
        ...

    @property
    def project_version(self) -> str:
        ...

    def close(self) -> None:
        ...

    def __enter__(self) -> MprReader:
        return self

    def __exit__(self, *args: object) -> None:
        self.close()
