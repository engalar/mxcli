from __future__ import annotations
import os
from pathlib import Path
from typing import Iterator

from mprutil.domain.unit import Unit, UnitId


class MprV2Reader:
    """Filesystem-backed MPR v2 reader (mprcontents/)"""

    def __init__(self, path: str) -> None:
        self._path = Path(path)
        if self._path.suffix == ".mpr":
            self._mprcontents = self._path.parent / "mprcontents"
        else:
            self._mprcontents = self._path
        if not self._mprcontents.is_dir():
            raise NotADirectoryError(f"mprcontents not found: {self._mprcontents}")

    def iter_units(self, type_filter: str | None = None) -> Iterator[Unit]:
        if not self._mprcontents.is_dir():
            return
        for root, _dirs, files in os.walk(self._mprcontents):
            for fn in files:
                if not fn.endswith(".mxunit"):
                    continue
                fpath = Path(root) / fn
                try:
                    data = fpath.read_bytes()
                except OSError:
                    continue
                if type_filter and type_filter not in data:
                    continue
                uid = fn[:-7]
                yield Unit(
                    id=UnitId(uid),
                    container_id=None,
                    containment_name="",
                    raw_bytes=data,
                )

    def get_unit(self, uid: str | UnitId) -> Unit | None:
        if isinstance(uid, UnitId):
            uid_str = uid.uuid
        else:
            uid_str = uid
        prefix = uid_str[:2]
        fpath = self._mprcontents / prefix / uid_str / f"{uid_str}.mxunit"
        alt = self._mprcontents / prefix / f"{uid_str}.mxunit"
        for p in [fpath, alt]:
            if p.is_file():
                data = p.read_bytes()
                return Unit(
                    id=UnitId(uid_str),
                    container_id=None,
                    containment_name="",
                    raw_bytes=data,
                )
        return None

    @property
    def project_version(self) -> str:
        return "?"

    def close(self) -> None:
        pass

    def __enter__(self) -> MprV2Reader:
        return self

    def __exit__(self, *args: object) -> None:
        self.close()
